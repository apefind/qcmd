
QCmd
====

QCmd is a lightweight Go-based terminal menu that lets you run shell commands
from a hierarchical, text-defined menu.

It’s designed to be fast, simple, and easy to customize using a single
``.qcmd`` file.

Features
--------

- Interactive terminal UI
- Simple indentation-based menu structure
- Nested submenus
- Run arbitrary shell commands
- Optional “return to menu” behavior
- Cross-platform (Linux, macOS, Windows)

Installation
------------

Run directly:

.. code-block:: sh

   go run qcmd.go

Install as a binary:

.. code-block:: sh

   go install qcmd.go

Usage
-----

By default, QCmd reads a file named ``.qcmd`` in the current directory:

.. code-block:: sh

   qcmd

Specify a custom file:

.. code-block:: sh

   qcmd -f path/to/file.qcmd

.qcmd File Format
-----------------

- Indentation defines menu hierarchy
- Lines ending with ``:`` define submenus
- ``label: command`` executes a shell command
- ``␍`` at the end of a line returns to the menu instead of exiting
- Comments start with ``#``
- Optional indentation directive:

  - ``#tab=4`` or ``#indent=2``

Example
-------

.. code-block:: text

   Go Tasks:
       go run qcmd.go
       go install qcmd.go
       go mod tidy
   Git:
       git commit -a && git push
       git commit -a
       Git Subcmds:
           git push ␍
           git pull ␍
           git deploy ␍
   ls -lart
   uname -a
   Print current directory: pwd ␍
   Edit .qcmd: $EDITOR .qcmd

Controls
--------

- Arrow keys / typing to select
- Enter to execute
- Esc or Ctrl+C to quit
