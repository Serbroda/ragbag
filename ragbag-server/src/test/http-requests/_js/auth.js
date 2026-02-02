const user = request.environment.get('auth_user');
const pass = request.environment.get('auth_pass');

const userpass = `${user}:${pass}`;
console.debug(userpass)

const base64auth = btoa(userpass);
request.variables.set("AUTH_HEADER", `Basic ${base64auth}`);
