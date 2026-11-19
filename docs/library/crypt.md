# `crypt` — hashes, encodings, passwords and encryption

`import crypt`

Hashes, encodings, passwords and encryption: fingerprints of text or files, Base64 and hex, safe password storage, and boxes locked with a passphrase.


## Functions

`hash`, `filehash`, `hmac`, `encode`, `decode`, `uuid`, `token`, `passwordhash`, `passwordcheck`, `encrypt`, `decrypt`.

### `hash[text, algorithm]`

A fingerprint of the text: the same text always gives the same hash. algorithm  optional: "sha256" (the default), "sha512", "sha1" or "md5" (sha1 and md5 only to match old systems) Gives back: hex text.

**Example**

```turtle
h = hash["hello"]
h = "hello" hash "sha512"
```

**Output**

```text
(prints nothing)
```

### `filehash[path, algorithm]`

hash of a file's contents (a checksum), read a piece at a time.

**Example**

```turtle
[write] release.zip
"release"
[end]
sum = filehash["release.zip"]
show length of sum .
```

**Output**

```text
64
```

### `hmac[text, key, algorithm]`

A signature: the hash of text with a secret key, as web APIs and webhooks use. algorithm as for hash.

**Example**

```turtle
body = "hello"
secret = "key"
sig = body hmac secret
show length of sig .
```

**Output**

```text
64
```

### `encode[text, how]`

how: "base64" (the default), "base64url" (for links, no padding) or "hex".

**Example**

```turtle
b = encode["hi"]
```

**Output**

```text
(prints nothing)
```

### `decode[text, how]`

The other way; how as for encode.

**Example**

```turtle
s = decode["aGk="]
```

**Output**

```text
(prints nothing)
```

### `uuid[]`

A random id (a version 4 UUID): 36 characters, different every time.

**Example**

```turtle
u = uuid[]
show length of u .
```

**Output**

```text
36
```

### `token[length]`

Random letters and digits from the system's secure source, for keys, session ids and reset links. length: 32 if left out (1 to 4096).

**Example**

```turtle
key = token[]
```

**Output**

```text
(prints nothing)
```

### `passwordhash[password]`

What to store instead of a password: salted PBKDF2-SHA256 (600,000 rounds), different each time for the same password.

**Example**

```turtle
pw = "secret"
stored = passwordhash[pw]
typed = "secret"
if ] passwordcheck[typed, stored] [
    show "password accepted" .
if [end]
```

**Output**

```text
password accepted
```

### `passwordcheck[password, stored]`

Whether password is the one stored was made from.

**Example**

```turtle
pw = "secret"
stored = passwordhash[pw]
show passwordcheck["wrong", stored] .
show passwordcheck["secret", stored] .
```

**Output**

```text
false
true
```

### `encrypt[text, passphrase]`

Locks text with a passphrase (AES-256-GCM, the key made from the passphrase with PBKDF2).

**Example**

```turtle
notes = "my note"
phrase = "pass"
box = encrypt[notes, phrase]
notes = decrypt[box, phrase]
show notes .
```

**Output**

```text
my note
```

### `decrypt[box, passphrase]`

Opens what encrypt made.

**Example**

```turtle
notes = "my note"
phrase = "pass"
box = encrypt[notes, phrase]
notes = decrypt[box, phrase]
show notes .
```

**Output**

```text
my note
```
