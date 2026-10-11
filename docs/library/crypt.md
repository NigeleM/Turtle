# `crypt` — hashes, encodings, passwords and encryption

[Library index](index.md) · `import crypt`

Hashes and HMACs, base64 and hex, ids and tokens, password hashing, and encryption with a passphrase.


`import crypt`: hashes, signatures, encodings, random ids and tokens,
passwords and encryption, on Go's standard library. Bad input (text that
isn't base64, a wrong passphrase, an unknown algorithm) is an error of
kind `crypt`.

| Function | Args | Returns |
|---|---|---|
| `hash` | text [, algorithm] | the text's fingerprint as hex. `"sha256"` (the default), `"sha512"`, or `"sha1"` / `"md5"` only to match old systems |
| `filehash` | path [, algorithm] | the hash of a file's contents (a checksum), read a piece at a time |
| `hmac` | text, key [, algorithm] | a signature: the hash of the text with a secret key, as hex, as web APIs and webhooks use |
| `encode` | text [, how] | `"base64"` (the default), `"base64url"` (for links, no padding) or `"hex"` |
| `decode` | text [, how] | the other way |
| `uuid` | — | a random id (version 4), 36 characters |
| `token` | [length] | random letters and digits from the system's secure source (32 if left out), for keys, session ids, reset links |
| `passwordhash` | password | what to store instead of a password: salted PBKDF2-SHA256 with 600,000 rounds; different each time |
| `passwordcheck` | password, stored | `true` if the password is the one `stored` was made from (compared in constant time) |
| `encrypt` | text, passphrase | the text locked with AES-256-GCM, its key made from the passphrase; text starting `turtle1:`, different each time |
| `decrypt` | box, passphrase | the text back; a wrong passphrase, or a box that was changed, is an error |

Each takes the sentence form too: `"hello" hash "sha512"`, `body hmac
secret`, `text encode "hex"`.

```
import crypt

show hash["hello"] .                    // 2cf24dba5fb0a30e...
sig = body hmac secret                   // check a webhook's signature
if ] sig != signature [
    fail "bad signature"
if [end]

stored = passwordhash[password]          // keep this, never the password
if ] passwordcheck[typed, stored] [
    show "welcome" .
if [end]

box = encrypt["the notes", phrase]       // "turtle1:..."
notes = decrypt[box, phrase]

id = uuid[]                              // "3f2a9c1e-...-4..."
key = token[]                            // 32 random letters and digits
b = encode["hi"]                         // "aGk="
```

Passwords: store only what `passwordhash` gives, and check with
`passwordcheck`; a plain `hash` of a password is easy to crack. `md5`
and `sha1` are broken for security; use them only to match an existing
system's checksums.

---

Copyright 2017-2026 Nigele McCoy. Licensed under the
[Apache License 2.0](../../LICENSE); see [NOTICE](../../NOTICE).
