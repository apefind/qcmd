QCmd
====
Run shell commands from .qcmd file.

Installation & usage::

    go install qcmd
    qcmd
    qcmd -f $HOME/.qcmd

Selection::

    Main Menu
    ┃ > ▸ Go Tasks
    ┃   ▸ Git
    ┃   ▶ ls -lart
    ┃   ▶ uname -a
    ┃   ▶ Print current directory
    ┃   ▶ Edit .qcmd

    Main Menu › Git
    ┃ > ▶ git commit -a && git push
    ┃   ▶ git commit -a
    ┃   ▸ Git Subcmds

Example .qcmd file::

    # .qcmd example file
    Go Tasks:
        go run qcmd.go
        go install qcmd.go
        go mod tidy
    Git:
        git commit -a && git push
        git commit -a
        Git Subcmds:
            git push
            git pull
            git deploy
    ls -lart
    uname -a
    Print current directory: pwd ␍
    Edit .qcmd: $EDITOR .qcmd
