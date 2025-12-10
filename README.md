Kbdctl
======

Currently supported keyboards:

- Zuoya GMK87 (time synch only)

## Zuoya GMK87

Problem: the supplied image and time synchronisation tool only works on
Windows.

[Jochen Eisinger][2] did a great job reverse engineering the protocol and
creating a [python script][1] for configuring the display GIF and
synchronising time.  USB interaction in this project is basically 1-to-1
rewrite in Go, everything in `kbd` package, so far, is a derived work from a
BSD licensed [zuoya_gmk87.py][1] script, (c) Copyright 2025 Jochen Eisinger.

This project's main purpose is to have a single binary for time synchronisation. 
If you need to upload GIFs into the keyboard, use the [original script][1].

[1]: https://gist.github.com/jeisinger/b198a72c5d7d203541c6269508c9ad8c
[2]: https://gist.github.com/jeisinger
