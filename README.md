# faxd

Daemon that listens to your email (via IMAP) and prints out any attachments
from a specified whitelist of receivers to whatever printer is configured by
default to `lp`.

Weirdest thing right now is that you need to configure login to email.
For Gmail, this means setting 2FA and then generating an "app password."

This is 90% vibe-coded. Check out `Plan.md` for implementation details.

