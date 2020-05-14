# possessions

[![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://github.com/volatiletech/possessions/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/volatiletech/possessions?status.svg)](https://godoc.org/github.com/volatiletech/possessions)
[![Go Report Card](https://goreportcard.com/badge/volatiletech/possessions)](http://goreportcard.com/report/volatiletech/possessions)

## Available Session Storers

* Disk
* Memory
* Redis
* Cookie

## Overseer interface

The job of an Overseer is to interface with your storers and manage your session
cookies. If you want to work with the session values you probably want to use
the higher level API functions listed above, but if you want to work with the
session directly you can use these overseer methods.

## Storer interface

In the case of session management, "keys" here are synonymous with session IDs. 
This is as lower level API and generally not used too much, because the above 
higher-level API functions call these functions. This can be used if you need
to deal with the storer directly for whatever reason.

## Available Overseers

```golang
// StorageOverseer is used for all server-side sessions (disk, memory, redis, etc).
NewStorageOverseer(opts CookieOptions, storer Storer) *StorageOverseer

//CookieOverseer is used for client-side only cookie sessions.
NewCookieOverseer(opts CookieOptions, secretKey [32]byte) *CookieOverseer
```

## How does each Storer work?

### Disk

Disk sessions store the session as a text file on disk. By default they store in 
the systems temp directory under a folder that is randomly generated when you 
generate your app using abcweb app generator command. The file names are the UUIDs 
of the session. Each time the file is accessed (using Get, Set, manually on 
disk, or by using the ResetMiddleware) it will reset the access time of the file, 
which will push back the expiration defined by maxAge. For example, if your
maxAge is set to 1 week, and your cleanInterval is set to 2 hours, then every 2
hours the cleaner will find all disk sessions files that have not been accessed 
for over 1 week and delete them. If the user refreshes a website and you're using
the ResetMiddleware then that 1 week timer will be reset. If your maxAge and 
cleanInterval is set to 0 then these disk session files will permanently persist, 
however the browser will still expire sessions depending on your cookieOptions 
maxAge configuration. In a typical (default) setup, cookieOptions will be set to 
maxAge 0 (expire on browser close), your DiskStorer maxAge will be set to 2 days,
and your DiskStorer cleanInterval will be set to 1 hour.

### Memory

Memory sessions are stored in memory in a mutex protected map[string]memorySession.
The key to the map is the session ID and the memorySession stores the value and expiry
of the session. The memory storer also has methods to start and stop a cleaner
go routine that will delete expired sessions on an interval that is defined when 
creating the memory session storer (cleanInterval).

### Redis

Redis sessions are stored in a Redis database. Different databases can be used
by specifying a different database ID on creation of the storer. Redis handles
session expiration automatically.

### Cookie

The cookie storer is intermingled with the CookieOverseer, so to use it you must
use the CookieOverseer instead of the StorageOverseer. Cookie sessions are stored
in encrypted form (AES-GCM encrypted and base64 encoded) in the clients browser.

## Middlewares

TODO: Document RefreshMiddleware

## Error types

If an API operation fails, and you would like to check if it failed due to no session
existing (errNoSession type), or the key used in a map-key operation not existing
(errNoMapKey type), you can check the error types using the following functions 
against the errors returned:

```golang
// errNoSession is a possible return value of Get and ResetExpiry
IsNoSessionError(err error) bool

// errNoMapKey is a possible return value of possessions.Get and possessions.GetFlash
// It indicates that the key-value map stored under a session did not have the 
// requested key
IsNoMapKeyError(err error) bool
```

## Examples

TODO: Add examples
