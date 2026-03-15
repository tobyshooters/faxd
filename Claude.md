How we like to write code.

Minimal indirection

Keep everything in one file as long as possible.

Only use helper functions if they're used more than once.

If a constraint can be added which simplifies the code, add it. It is better
to have a tighter area of operation than code which is very permissive.

Keep defensive code on the boundaries. E.g. validate user input where the HTTP
request is handled. Once inside the core of the app, assume everything is
well-defined and correctly formatted.

To enforce the previous constraint, avoid types which are optional, or
possibly null, or possibly undefined. This makes code which is ugly.

Try to write "brutalist" code.

Avoid long lines of code. Prefer to break a statement into smaller
sub-statements with good variable names.

Do not be too verbose. Prefer short variable names to long ones. Do not repeat
prefixes when they can be infered from the context.

For CSS files, write one property per line.
Our goal is legibility and simplicity, not unnecessary terseness.
