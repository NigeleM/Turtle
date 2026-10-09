-- Turtle.app: double-click a .turtle file to run it in Terminal; open the
-- app itself for the Turtle prompt. Built by build-pkg.sh.

on open theFiles
	repeat with f in theFiles
		do shell script "/usr/local/share/turtle/turtle-run " & quoted form of (POSIX path of f)
	end repeat
end open

on run
	do shell script "open -a Terminal /usr/local/bin/turtle"
end run
