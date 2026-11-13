# Datenschutzerklärung

<!-- Vorlage für eine Ragbag-Instanz. Ersetze alles in […] und streiche Abschnitte, die nicht zutreffen
     (z. B. E-Mail ohne SMTP_HOST, Reverse Proxy, Backups). Keine Rechtsberatung. -->

## 1. Verantwortlicher

[Vorname Nachname]\
[Anschrift]\
E-Mail: [kontakt@example.com]

Diese Erklärung beschreibt, welche Daten der Dienst „Ragbag“ unter [https://ragbag.example.com] verarbeitet.
Ragbag ist eine selbst betriebene Software zum gemeinsamen Sammeln von Links. Es gibt keine Werbung, kein
Tracking und keine Analyse-Dienste.

## 2. Hosting

Der Dienst läuft auf einem Server von [Hosting-Anbieter, Anschrift] in [Land]. Mit dem Anbieter besteht ein
Vertrag zur Auftragsverarbeitung nach Art. 28 DSGVO.
<!-- Läuft der Server bei dir zu Hause, ersetze den Absatz: "Der Dienst läuft auf einem Server des
     Verantwortlichen." -->

## 3. Aufruf der Seiten und Server-Protokolle

Bei jedem Aufruf verarbeitet der Server technisch notwendig deine IP-Adresse, Datum und Uhrzeit, die
aufgerufene Adresse (ohne Suchparameter; geheime Teile wie Freigabe-Tokens werden unkenntlich gemacht), den
Statuscode, die Antwortzeit und, wenn du angemeldet bist, deinen Benutzernamen. Diese Protokolle dienen dem
sicheren Betrieb, der Fehlersuche und der Abwehr von Angriffen (z. B. erratene Passwörter). Sie werden nach
[7] Tagen gelöscht.
<!-- Stimmt nur, wenn es auch so eingestellt ist: LOG_FILE mit LOG_MAX_AGE_DAYS=7, oder journald mit
     MaxRetentionSec=7day; bei Docker begrenzen die Compose-Dateien die Größe, nicht die Dauer. Mit
     LOG_ANONYMIZE_IP=true ergänze: "Die IP-Adresse wird dabei gekürzt gespeichert (ohne das letzte Byte)." --> Rechtsgrundlage ist Art. 6 Abs. 1 lit. f DSGVO; unser berechtigtes Interesse ist ein
sicherer und stabiler Dienst.

Zum Schutz vor Missbrauch zählt der Server Anfragen je IP-Adresse und Konto (Rate-Limits). Diese Zähler liegen
nur im Arbeitsspeicher und sind nach kurzer Zeit oder einem Neustart weg.

## 4. Konto

Für ein Konto speichern wir Benutzernamen, E-Mail-Adresse, Passwort (nur als BCrypt-Hash), optional einen
Anzeigenamen, ein Profilbild und eine PIN für gesperrte Sammlungen (ebenfalls nur als Hash), den Zeitpunkt der
Registrierung und der Zustimmung zu den Nutzungsbedingungen. Rechtsgrundlage ist die Erfüllung des
Nutzungsvertrags (Art. 6 Abs. 1 lit. b DSGVO).

Schaltest du die Zwei-Faktor-Anmeldung ein, speichern wir den Schlüssel für deine Authenticator-App
verschlüsselt und die Wiederherstellungscodes nur als Hash. Für API-Tokens speichern wir Name, Berechtigung,
Ablaufdatum, den Zeitpunkt der letzten Nutzung und einen Hash des Tokens; das Token selbst siehst nur du, einmal
beim Erstellen.

<!-- Nur mit SMTP_HOST: -->
Zur Bestätigung der E-Mail-Adresse und zum Zurücksetzen des Passworts schicken wir dir E-Mails mit einem Link;
gespeichert wird nur ein Hash des Links (48 bzw. 1 Stunde gültig). Konten, die nicht innerhalb von 7 Tagen
bestätigt werden, löschen wir automatisch. Die E-Mails versendet [E-Mail-Anbieter, Anschrift] in unserem Auftrag.

Du kannst dein Konto jederzeit selbst löschen (Profil → Delete account). Dabei werden Name, E-Mail-Adresse,
Passwort, Profilbild, PIN, Favoriten, Reaktionen und Dashboards gelöscht, ebenso Spaces, in denen nur du Mitglied
bist. Links, die du in Spaces mit anderen Mitgliedern eingetragen hast, bleiben dort ohne deinen Namen erhalten,
weil sie zum gemeinsamen Inhalt des Space gehören.

## 5. Inhalte

Spaces, Sammlungen, Links mit Titel, Beschreibung, Tags und Vorschaubild, Reaktionen und Favoriten sehen die
Mitglieder des jeweiligen Space; private Sammlungen nur ihre Mitglieder. Gelöschtes liegt 30 Tage im Papierkorb
und wird danach endgültig gelöscht. Wer einen öffentlichen Link zu einer Sammlung erstellt, macht diese Sammlung
für alle mit dem Link sichtbar – ohne Mitglieder und ohne wer was eingetragen hat.

Wenn du einen Link speicherst, ruft der Server die Seite auf, um Titel, Beschreibung und Vorschaubild zu lesen.
Diese Anfrage kommt vom Server, nicht von deinem Gerät. Ebenso lädt der Server die Nachrichten-Feeds (RSS
oder Atom), die an Links hinterlegt sind, um Schlagzeilen anzuzeigen; er hält sie bis zu einer Stunde im
Arbeitsspeicher und speichert nichts davon.

## 6. Inhalte von anderen Seiten

Vorschaubilder und Symbole (Favicons) gespeicherter Links und von Schlagzeilen lädt dein Browser direkt von den
jeweiligen Seiten. Dabei erfahren deren Betreiber deine IP-Adresse und den Zeitpunkt, aber nicht, dass du Ragbag
nutzt (es wird keine Herkunftsadresse mitgeschickt). Wer das nicht möchte, kann Vorschaubilder je Link ausblenden.

## 7. Discord

Wer einen Space verwaltet, kann ihn mit einem Discord-Kanal verbinden. Dann schickt der Server zu jedem neuen
Link Titel, Adresse, Beschreibung und Vorschaubild, den Namen der Person, die ihn eingetragen hat, sowie den
Namen der Sammlung und des Space an Discord (Discord Inc., 444 De Haro Street, Suite 200, San Francisco, CA 94107,
USA). Dort sieht es, wer den Kanal lesen kann; es gilt die Datenschutzerklärung von Discord. Links in privaten
Sammlungen und Importe werden nie geschickt. Rechtsgrundlage ist das berechtigte Interesse der Mitglieder, neue
Links dort zu sehen, wo sie sich ohnehin austauschen (Art. 6 Abs. 1 lit. f DSGVO).

## 8. Speicher im Browser

Ragbag setzt ein technisch notwendiges Cookie für die Anmeldung (`refresh_token`, nur für die Anmeldung, bis zu
[14] Tage). Im lokalen Speicher des Browsers liegen das Anmelde-Token und deine Anzeige-Einstellungen
(Ansicht, Sortierung, zuletzt geöffneter Space, schon gesehene Schlagzeilen, geschlossene Hinweise), für die
Dauer der Sitzung außerdem das Token für entsperrte Sammlungen und ein noch nicht gespeicherter Link oder eine
Einladung, bis du angemeldet bist. Das ist für den Dienst erforderlich (§ 25 Abs. 2 Nr. 2 TDDDG);
eine Einwilligung ist dafür nicht nötig. Es gibt keine Cookies von Dritten.

## 9. Meldungen

Öffentliche Links kann jede Person melden. Wir speichern dann die Meldung, optional Name und E-Mail-Adresse der
meldenden Person (um über die Entscheidung zu informieren) und die Entscheidung. Erledigte Meldungen löschen wir
nach 180 Tagen. Rechtsgrundlage sind unsere Pflichten nach Art. 16 und 17 DSA (Art. 6 Abs. 1 lit. c DSGVO).

## 10. Datensicherung

<!-- Streichen, wenn du keine Backups machst. -->
Zur Sicherung erstellen wir regelmäßig Backups der Datenbank, die [7] Tage aufbewahrt werden
<!-- BACKUP_KEEP × BACKUP_INTERVAL_HOURS, plus Kopien an anderen Orten -->. Gelöschte Daten verschwinden mit dem
Ablauf der Backups.

## 11. Deine Rechte

Du hast das Recht auf Auskunft (Art. 15 DSGVO), Berichtigung (Art. 16), Löschung (Art. 17), Einschränkung der
Verarbeitung (Art. 18), Datenübertragbarkeit (Art. 20; Sammlungen lassen sich als JSON exportieren) und
Widerspruch gegen Verarbeitungen auf Grundlage berechtigter Interessen (Art. 21). Schreib dazu an
[kontakt@example.com].

Du kannst dich außerdem bei einer Datenschutz-Aufsichtsbehörde beschweren, z. B. bei der für uns zuständigen:
[Name und Anschrift der Landesdatenschutzbehörde].

Stand: [Datum]
