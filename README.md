Hoot
====

## Installing

This app lives at the package url `github.com/chrismckenzie/hoot`. compiling will
require all source be placed at that path in your `$GOPATH`.

## Running

Hoot when starting up will immediately look for a json config file in the path
given on the flag `-config` if no file path is given it will simple startup
using a default configuration defined here

```
{
  "port": ":9000",
  "logpath": "./hoot.log"
}
```

hoot will also default any fields left blank to the same values mentioned above.

## Implementation

Hoot keeps each users tcp connection with the user data and
sets up a handler goroutine for processing of messages if
the incoming message contains a slash it will be parsed as
a user command (run /help to see available commands).

Users can be a member of one "Room" at a time and all messages
that are not user commands will be sent to all users belonging to 
that "Room". when a user joins a room they will automatically be
"caughtup" and all previous messages will be sent to the joining user.

## Known Issues

- There is a small UI issue where if a user is writing a message and
another sends a message the cursor for the receiving user will be reset 
back to the home position. this is due to VT100 and compatible terminals
not support return to cursor in a reliable way.
