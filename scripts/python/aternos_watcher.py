from typing import Dict, List, Any, Union, Hashable

import yaml
import time
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.expected_conditions import _find_element
from selenium.webdriver.chrome.options import Options
from enum import Enum

WEB_WAIT_TIME = 60 * 3

app_properties = None
driver = None
queue_confirmed = False


class ServerState(Enum):
    OFFLINE = 1,
    ONLINE = 2,
    STARTING = 3,
    STOPPING = 4,
    LOADING = 5,
    QUEUING = 6,
    SAVING = 7,
    UNKNOWN = 8


class AppProperties:
    def __init__(self, driver_path: str, base_url: str, user: str, password: str, personal_page: str,
                 headless: bool = True, binary_path: str = None):
        self.driver_path = driver_path
        self.base_url = base_url
        self.user = user
        self.password = password
        self.personal_page = personal_page
        self.headless = headless
        self.binary_path = binary_path


class text_to_change(object):
    def __init__(self, locator, text):
        self.locator = locator
        self.text = text

    def __call__(self, driver):
        actual_text = _find_element(driver, self.locator).text
        return actual_text != self.text


def find_nested_property(cfg: Union[Union[Dict[Hashable, Any], List[Any], None], Any], prop_name: str):
    splitted_props = prop_name.split('.')
    tmp_prop = None
    for i in range(len(splitted_props)):
        if i == 0:
            tmp_prop = cfg.get(splitted_props[i])
        else:
            if tmp_prop is None:
                break
            tmp_prop = tmp_prop.get(splitted_props[i])
    return tmp_prop


def required_prop(cfg: Union[Union[Dict[Hashable, Any], List[Any], None], Any], prop_name: str):
    val = find_nested_property(cfg, prop_name)
    if val is None:
        raise AttributeError("Property %s is required but not set" % prop_name)
    return val


def default_prop(cfg: Union[Union[Dict[Hashable, Any], List[Any], None], Any], prop_name: str, default_val):
    val = find_nested_property(cfg, prop_name)
    if val is None:
        return default_val
    else:
        return val


def load_config() -> AppProperties:
    with open("config.yml", 'r') as stream:
        try:
            cfg = yaml.safe_load(stream)
            return AppProperties(
                driver_path=required_prop(cfg, 'selenium.chrome-driver-path'),
                base_url=default_prop(cfg, 'http-base-url', "https://aternos.org"),
                user=required_prop(cfg, 'auth.user'),
                password=required_prop(cfg, 'auth.password'),
                personal_page=required_prop(cfg, 'http.personal-page'),
                headless=default_prop(cfg, 'selenium.headless', True),
                binary_path=default_prop(cfg, 'selenium.chrome-binary-path', None)
            )
        except yaml.YAMLError as exc:
            raise exc


def determine_server_status(status_text: str) -> ServerState:
    status_text = status_text.lower()
    if status_text == "online" or "online" in status_text:
        return ServerState.ONLINE
    elif status_text == "offline" or "offline" in status_text:
        return ServerState.OFFLINE
    elif status_text == "starting ..." or "starting" in status_text:
        return ServerState.STARTING
    elif status_text == "stopping ..." or "stopping" in status_text:
        return ServerState.STOPPING
    elif status_text == "loading ..." or "loading" in status_text:
        return ServerState.LOADING
    elif status_text == "waiting in queue" or "queue" in status_text:
        return ServerState.QUEUING
    elif status_text == "saving ..." or "saving" in status_text:
        return ServerState.SAVING
    else:
        return ServerState.UNKNOWN


def go_if_necessary(url: str):
    if driver.current_url != url:
        driver.get(url)
        time.sleep(1)


def is_element_interactible(element: WebElement) -> bool:
    return element.is_displayed() and element.is_enabled()


def is_element_by_id_visible(id: str):
    try:
        el = driver.find_element_by_id(id)
        return True, el, is_element_interactible(el)
    except:
        return False, None, False


def is_element_by_css_selector_visible(css: str):
    try:
        el = driver.find_element_by_css_selector(css)
        return True, el, is_element_interactible(el)
    except:
        return False, None, False


def set_language(lang: str):
    go_if_necessary("%s/language/%s/" % (app_properties.base_url, lang))


def is_logged_in() -> bool:
    go_if_necessary("%s/server/" % app_properties.base_url)
    input_user = is_element_by_id_visible("user")
    input_password = is_element_by_id_visible("password")
    login_submit = is_element_by_id_visible("login")
    return not input_user[0] and not input_password[0] and not login_submit[0]


def login(username: str, password: str):
    print("Logging in as %s..." % username)

    if is_logged_in():
        print("Already logged in")
        return

    go_if_necessary("%s/server/" % app_properties.base_url)

    input_user = driver.find_element_by_id("user")
    input_password = driver.find_element_by_id("password")
    login_submit = driver.find_element_by_id("login")

    input_user.clear()
    input_user.send_keys(username)

    input_password.clear()
    input_password.send_keys(password)

    login_submit.click()
    time.sleep(1)


def get_server_state_personal_page() -> ServerState:
    go_if_necessary(app_properties.personal_page)

    el = is_element_by_css_selector_visible("div.status")
    if el[0]:
        return determine_server_status(el[1].text)
    else:
        return ServerState.UNKNOWN


def get_server_state_server_page() -> str:
    go_if_necessary("%s/server/" % app_properties.base_url)

    el = is_element_by_css_selector_visible("div.server-status span.statuslabel-label")
    if el[0]:
        return determine_server_status(el[1].text)
    else:
        return ServerState.UNKNOWN


def get_queue_time_seconds() -> int:
    go_if_necessary("%s/server/" % app_properties.base_url)
    el = is_element_by_css_selector_visible("div.statuslabel span.server-status-label-left.queue-time")
    if el[0]:
        whitelist = set('0123456789')
        try:
            return int(''.join(filter(whitelist.__contains__, el[1].text))) * 3 * 60
        except:
            return WEB_WAIT_TIME
    else:
        return WEB_WAIT_TIME


def handle_server_state_action() -> bool:
    global queue_confirmed
    status = get_server_state_server_page()

    print("Handling step. Current server state is %s" % status)

    start_button = is_element_by_css_selector_visible("div#start")
    notifications_cancel_button = is_element_by_css_selector_visible(
        "div.alert > main > div.alert-buttons a.btn.btn-separate.btn-red")

    if notifications_cancel_button[0] and notifications_cancel_button[2]:
        print("Declining notification dialog...")
        notifications_cancel_button[1].click()
        return False
    elif start_button[0] and start_button[2]:
        print("Clicking start button...")
        start_button[1].click()
        return False
    elif status == ServerState.ONLINE or status == ServerState.STARTING:
        print("Server status is %s. No need to do anything..." % status)
        return True
    elif status == ServerState.LOADING or status == ServerState.STOPPING or status == ServerState.SAVING or (
            status == ServerState.QUEUING and queue_confirmed):
        print("Server state is in idle mode. Waiting for changes...")
        text = "Loading ..."
        current_text_element = is_element_by_css_selector_visible("div.server-status span.statuslabel-label")
        if current_text_element[0]:
            text = current_text_element[1].text
        WebDriverWait(driver, WEB_WAIT_TIME).until(
            text_to_change((By.CSS_SELECTOR, "div.server-status span.statuslabel-label"), text)
        )
        return False
    elif status == ServerState.QUEUING:
        print("Waiting in queue for confirm button gets visible...")
        confirm_button = WebDriverWait(driver, get_queue_time_seconds()).until(
            EC.element_to_be_clickable((By.ID, "confirm"))
        )
        if is_element_interactible(confirm_button):
            confirm_button.click()
            queue_confirmed = True
        return False
    else:
        return True


def start_server():
    print("Starting server...")

    go_if_necessary("%s/server/" % app_properties.base_url)

    scnmcp_button = is_element_by_id_visible("sncmp-popup-ok-button")
    if scnmcp_button[0] and scnmcp_button[2]:
        scnmcp_button[1].click()

    eula_div = is_element_by_css_selector_visible("div.alert main > div.alert-buttons > a.btn.btn-separate.btn-green")
    if eula_div[0] and eula_div[2]:
        eula_div[1].click()

    while not handle_server_state_action():
        time.sleep(1)

    print("Successfully start server starting process")


def main():
    global app_properties
    global driver
    
    try:
        app_properties = load_config()

        chrome_options = Options()
        if app_properties.headless:
            chrome_options.add_argument("--headless")
        if app_properties.binary_path is not None:
            chrome_options.binary_location = app_properties.binary_path

        driver = webdriver.Chrome(executable_path=app_properties.driver_path, options=chrome_options)

        set_language("en")

        if get_server_state_personal_page() == ServerState.ONLINE:
            print("Server is already running")
        else:
            login(app_properties.user, app_properties.password)
            start_server()

    finally:
        if driver is not None:
            driver.quit()


if __name__ == '__main__':
    # execute only if run as the entry point into the program
    main()
