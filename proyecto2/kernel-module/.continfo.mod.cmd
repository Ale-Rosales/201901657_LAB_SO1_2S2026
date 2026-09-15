savedcmd_continfo.mod := printf '%s\n'   continfo.o | awk '!x[$$0]++ { print("./"$$0) }' > continfo.mod
