# QCmd

**QCmd** is a lightweight terminal menu written in Go that allows you to run
shell commands from a simple text-defined menu.

Commands are organized hierarchically using indentation in a single
`.qcmd` configuration file.

QCmd is designed to be:

* fast
* minimal
* portable
* easy to customize

It works on **Linux**, **macOS**, and **Windows**.

## Quick Preview

.. image:: demo.gif
:alt: QCmd terminal demo
:align: center
:width: 800px

The demo shows navigating menus and executing commands directly from the
terminal interface.

## Features

* Interactive terminal UI
* Indentation-based menu structure
* Nested submenus
* Searchable command palette
* Breadcrumb navigation
* Visual separators
* Optional return-to-menu behavior
* Cross-platform shell execution
* Single-file configuration

## Installation

Run directly with Go:

.. code-block:: sh

go run qcmd.go

Install as a binary:

.. code-block:: sh

go install

After installation:

.. code-block:: sh

qcmd

## Usage

By default, QCmd reads a file named `.qcmd` in the current directory.

.. code-block:: sh

qcmd

Specify a custom configuration file:

.. code-block:: sh

qcmd -f path/to/file.qcmd

Open the **command palette** (search all commands):

.. code-block:: sh

qcmd -p

## Configuration File

QCmd reads commands from a `.qcmd` file.

The structure is defined using indentation.

Rules

```

- Indentation defines menu hierarchy
- Lines ending with ``:`` create submenus
- ``label: command`` executes a shell command
- ``---`` creates a visual separator
- ``␍`` at the end of a line returns to the menu after execution
- Blank lines are ignored
- Comments start with ``#``

Optional indentation directive:

.. code-block:: text

   #tab=4

or

.. code-block:: text

   #indent=2

Example Configuration
---------------------

Example ``.qcmd`` file:

.. code-block:: text

   #tab=4

   Go Tasks:
       Run: go run qcmd.go
       Install: go install
       Tidy modules: go mod tidy

   Git:
       Commit + Push: git commit -a && git push
       Commit: git commit -a
       ---
       Git Subcommands:
           Push: git push ␍
           Pull: git pull ␍
           Status: git status ␍

   ---
   List files: ls -lart
   System info: uname -a
   Print current directory: pwd ␍
   Edit config: $EDITOR .qcmd

Menu Example
------------

Example terminal interface:

.. code-block:: text

   QCmd
   ┃ > › Go Tasks
   ┃   › Git
   ┃   → List files
   ┃   → System info
   ┃   → Print current directory
   ┃   → Edit config

   QCmd › Git
   ┃ > → Commit + Push
   ┃   → Commit
   ┃   › Git Subcommands

Command Palette
---------------

The command palette allows fast fuzzy searching across all commands.

Run:

.. code-block:: sh

   qcmd -p

This is useful when you want to run a command quickly without navigating
through nested menus.

Controls
--------

Menu Navigation

- **Arrow keys** — move selection
- **Type to search** — filter entries
- **Enter** — execute selected command
- **Esc / Ctrl+C** — exit

Command Palette

- **Type** to filter commands
- **Enter** to execute
- **Esc / Ctrl+C** to cancel

Shell Execution
---------------

Commands are executed using the system shell.

Linux / macOS:

.. code-block:: text

   sh -c "command"

Windows:

- ``pwsh`` (PowerShell Core) if available
- otherwise ``powershell``

Command output is streamed directly to the terminal.

Project Structure
-----------------

::

   qcmd.go
   .qcmd
   README.rst
   demo.gif

License
-------

MIT License
```
