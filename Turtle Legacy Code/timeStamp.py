

# program used to tell what was the last file accessed in directory
import os
#print(os.listdir(os.getcwd()))

files = []
tfiles = []
#print(tfiles)
# Getting the current directory listing
files = os.listdir(os.getcwd())

# add files to tfiles list
# t-files are file extension for turtle interpreter to read
for file in files:
    if file.find(".t") > 0:
        tfiles.append(file)
    else:
        continue


#print(tfiles)
# Clever trick to pass in the double clicked t file
# Get the last time the file was modified
# add that time to lastAccess list
# find the greatest value and its index
# use that index to put into tfiles
# then pass that the name of file into os.system command to run file
# and open the file in the ide
count = 0

lastAccess = []
if len(tfiles) == 1:
    os.system("./turtle"+" "+tfiles[0])
else:
    while count != len(tfiles):
        access = os.stat(tfiles[count])
        lastAccess.append(access[9])
        count +=1

# Finding last modified file
count = max(lastAccess)
index = lastAccess.index(count)
os.system("./turtle "+str(tfiles[index]))


