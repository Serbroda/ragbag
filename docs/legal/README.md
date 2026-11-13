# Legal page templates

Drafts in German for an instance run in Germany, describing what Ragbag actually processes. They are
**templates, not legal advice**: fill in the `[…]` parts, remove what does not apply (e.g. email when there is no
`SMTP_HOST`) and have them checked if you are unsure.

Copy them into the data directory as `legal/imprint.md`, `legal/privacy.md` and `legal/terms.md`
(systemd: `/var/lib/ragbag/legal/`, Docker: `/data/legal/`, from source: `data/legal/`). Ragbag shows them at
`/legal/imprint` & co., links them on the sign-in page, on public links and in the account menu, and asks
new accounts to accept the terms. Changes apply without a restart. A first line `# Title` becomes the page title;
raw HTML is not shown (comments like the ones in the templates are dropped).

When the terms of use change, put the new text in place and record the change in the administration
(overview, “Terms changed…”): from the chosen day every account has to accept them again before going on
(API tokens answer 403 until then), a later day shows a notice first. Typos need no new acceptance; the
privacy policy is information and is never accepted (tell people about bigger changes anyway).

Pages that live elsewhere (e.g. the imprint of your website) are linked with `IMPRINT_URL`, `PRIVACY_URL` and
`TERMS_URL` instead.
