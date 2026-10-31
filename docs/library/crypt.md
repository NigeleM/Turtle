# `crypt` — hashes, encodings, passwords and encryption

[Library index](index.md) · `import crypt`

Hashes and HMACs, base64 and hex, ids and tokens, password hashing, and encryption with a passphrase.


`import crypt`: hashes, signatures, encodings, random ids and tokens,
passwords and encryption. Bad input (text that
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

> *Illustrative.* This example needs values that the example does not set, so it is shown for reading and is not run as a test.

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

<!-- generated: turtle doc `crypt` (regenerate with the script; do not edit by hand) -->
## Function reference

The entries below are copied from the binary's own documentation (`turtle doc crypt`), so they match the version you run.

```text
import crypt
```

### `Hashes, signatures, encodings, random ids and tokens, passwords and`

```text
Hashes, signatures, encodings, random ids and tokens, passwords and
encryption. Bad input (text that isn't base64, a wrong passphrase, an
unknown algorithm) is an error of kind crypt.
```

### `hash[text, algorithm]`

```text
hash[text, algorithm]        (import crypt)
  A fingerprint of the text: the same text always gives the same hash.
  algorithm  optional: "sha256" (the default), "sha512", "sha1" or "md5"
             (sha1 and md5 only to match old systems)
  Gives back: hex text.
  Example:
    h = hash["hello"]
    h = "hello" hash "sha512"
```

### `filehash[path, algorithm]`

```text
filehash[path, algorithm]        (import crypt)
  hash of a file's contents (a checksum), read a piece at a time.
  Example:
    [write] release.zip
    "release"
    [end]
    sum = filehash["release.zip"]
    show length of sum .
```

### `hmac[text, key, algorithm]`

```text
hmac[text, key, algorithm]        (import crypt)
  A signature: the hash of text with a secret key, as web APIs and
  webhooks use. algorithm as for hash.
  Gives back: hex text.
  Example:
    body = "hello"
    secret = "key"
    sig = body hmac secret
    show length of sig .
```

### `encode[text, how]`

```text
encode[text, how]        (import crypt)
  how: "base64" (the default), "base64url" (for links, no padding) or
  "hex".
  Example:
    b = encode["hi"]              // "aGk="
```

### `decode[text, how]`

```text
decode[text, how]        (import crypt)
  The other way; how as for encode.
  Example:
    s = decode["aGk="]            // "hi"
```

### `uuid[]`

```text
uuid[]        (import crypt)
  A random id (a version 4 UUID): 36 characters, different every time.
```

### `token[length]`

```text
token[length]        (import crypt)
  Random letters and digits from the system's secure source, for keys,
  session ids and reset links. length: 32 if left out (1 to 4096).
  Example:
    key = token[]
```

### `passwordhash[password]`

```text
passwordhash[password]        (import crypt)
  What to store instead of a password: salted PBKDF2-SHA256 (600,000
  rounds), different each time for the same password.
  Example:
    pw = "secret"
    stored = passwordhash[pw]
    typed = "secret"
    if ] passwordcheck[typed, stored] [
        show "password accepted" .
    if [end]
```

### `passwordcheck[password, stored]`

```text
passwordcheck[password, stored]        (import crypt)
  Whether password is the one stored was made from.
  Gives back: true or false.
  Example:
    pw = "secret"
    stored = passwordhash[pw]
    show passwordcheck["wrong", stored] .
    show passwordcheck["secret", stored] .
```

### `encrypt[text, passphrase]`

```text
encrypt[text, passphrase]        (import crypt)
  Locks text with a passphrase (AES-256-GCM, the key made from the
  passphrase with PBKDF2). Different each time; only decrypt with the
  same passphrase opens it.
  Gives back: text starting "turtle1:", safe to save or send.
  Example:
    notes = "my note"
    phrase = "pass"
    box = encrypt[notes, phrase]
    notes = decrypt[box, phrase]
    show notes .
```

### `decrypt[box, passphrase]`

```text
decrypt[box, passphrase]        (import crypt)
  Opens what encrypt made. A wrong passphrase, or a box that was
  changed, is an error.
  Example:
    notes = "my note"
    phrase = "pass"
    box = encrypt[notes, phrase]
    notes = decrypt[box, phrase]
    show notes .
```

<!-- end generated -->
