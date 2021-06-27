
# Genesis Programming language
# Genesis Language interpreter
# Created 2016


# varlist used to appends lines from the file that is being read
varlist = []
# Dictionary for the variables
vardict = {}
# things that are understood by the interpreter are append to clist
clist = []
# Count list
numberlist = []
# import libraies 
from sys import *
# re is regular expression
import re
# sys for sys.argv for command line use 
import sys
# import os for operating system functions such as directories and etc
import os
# import webbrowser to access internet through python
import webbrowser
# Import shutil for moving, renaming files and etc
import shutil



def open_file(filename):
      filename = ''
      filename = input('')
      # Reads the file name from the terminal
      if filename == '[shell]':
            Shell()
      else:
            return filename

def info(filename): # file name passed into info function
      
      lister = " "
      count = 0
      length = 0
      line = ""
      var = ""
      con = 0
      

      # Opening file for readind
      try:
            file = open(filename,'r')

      # reading lines from the file
      # appending the lines to varlist
      
            while lister != "":
                  lister = file.readline()
                  varlist.append(lister)
      
            file.close()

      except TypeError:
            pass
      

      length = len(varlist)
      # read and interpret lines until
      # count and length are equal
      while count != length:
            
            # Show statement 
            # string works
            line = varlist[count]
            try:  # Key word show is equal to print
                  var = re.search(r'show', line)
                  var = var.group(0)
                  if var == "show":
                        try:  # Search for string
                              var = re.search(r'".+"', line)
                              var = var.group(0)
                              if var != "\"":
                                    # Takes string quotes away
                                    var = var.replace("\"",'')
                                    clist.append(var)
                                    try: # looks for period
                                          var = re.search(r'\.', line)
                                          var = var.group(0)
                                          count = count + 1
                                    except AttributeError:
                                          clist.pop()
                                          clist.append("NO PERIOD")
                                          clist.append(count)
                                          var = "NO PERIOD"
                                          return var
                                          
                                    
                                    
                         
                        # Number works
                        # Digits do not need to be in a string
                        except AttributeError:
                              try:  # Digits
                                    var = re.search(r'\d.+', line)
                                    var = var.group(0)
                                    if var != "None":
                                          try:
                                                var = var.replace(" .",'')
                                                var = eval(var)
                                                clist.append(var)
                                                try:
                                                      var = re.search(r'\.', line)
                                                      var = var.group(0)
                                                      count = count + 1
                                                except AttributeError:
                                                      clist.pop()
                                                      clist.append("NO PERIOD")
                                                      clist.append(count)
                                                      var = "NO PERIOD"
                                                      return var
                                          except NameError:
                                                clist.append(var)
                                                try:
                                                      var = re.search(r'\.', line)
                                                      var = var.group(0)
                                                      count = count + 1
                                                except AttributeError:
                                                      clist.pop()
                                                      clist.append("NO PERIOD")
                                                      clist.append(count)
                                                      var = "NO PERIOD"
                                                      return var
                                                
                              # Variables work
                              except AttributeError:
                                    try:  # gets the variable name
                                          var = re.search(r'\s\w+\b', line)
                                          var = var.group(0)
                                          # cleans the variable name up
                                          var = var.replace(" ",'')
                                          var = var.replace(".",'')
                                          # looks for variable value in dictionary
                                          var = vardict.get(var)
                                          if var != 'None':
                                                clist.append(var)
                                                try: # looks for period
                                                      var = re.search(r'\.', line)
                                                      var = var.group(0)
                                                      count = count + 1
                                                except AttributeError:
                                                      clist.pop()
                                                      clist.append("NO PERIOD")
                                                      clist.append(count)
                                                      var = "NO PERIOD"
                                                      return var
                                          
                                    
                                    except AttributeError:
                                          var = "No var"
                                          clist.append(var)
                                          clist.append(count)
                                          return var
                                          
                        
            except AttributeError:
                  # Skips over line if \n
                  if varlist[count] == "\n" or varlist[count] == '\n':
                        count = count + 0
                        
                  # Variables 
                  elif varlist[count] != "\n" and varlist[count] != "":
                        try:
                              # reads variable name
                              var = re.search(r'\w+\b',line)
                              var = var.group(0)
                              if var != "None":
                                    info = ""
                                    # variable name append to info
                                    info += var
                                    
                                    vardict[info] = ""
                                    # find equals symbol
                                    var = re.search(r'\s=\s',line)
                                    var = var.group(0)
                                    if var != 'None':
                                          # Words and strings
                                          try:
                                                var = re.search(r'\W\b.+$',line)
                                                var = var.group(0)
                                                Varb = ""
                                                Varb += var
                                                Varb = Varb.replace(' ','')
                                                Varb = vardict.get(Varb)
                                                if Varb == None:
                                                      
                                                      if var != "None":
                                                            var = var.replace("\"",'')
                                                                  
                                                            # cleans value
                                                            try:
                                                                  # Test for expressions
                                                                  vardict[info] = var
                                                                  #if expression find its value
                                                                  var = eval(vardict.get(info))
                                                                  vardict[info] = var
                                                                  var = vardict.get(info)
                                                                  vardict.update()
                                                                  count = count + 1
                                                            except (NameError,SyntaxError) as error:
                                                                  vardict[info] = var
                                                                  var = vardict.get(info)
                                                                  vardict.update()
                                                                  count = count + 1
                                                elif Varb != None:
                                                      vardict[info] = Varb
                                                      count = count + 1

                                          # input variable 
                                          except AttributeError:
                                                try:
                                                      amount = 0
                                                      num = 0
                                                      total = len(clist)
                                                      var = re.search(r'\?',line)
                                                      var = var.group(0)
                                                      num = len(numberlist)
                                                      # The next few lines are for output
                                                      # After we test the clist and numberlist
                                                      # we will show output or display input
                                                      if num > 0:
                                                            numberlist.reverse()
                                                            amount = numberlist[0]
                                                            if var == '?':
                                                                  while amount != total:
                                                                        print(clist[amount])
                                                                        amount = amount + 1
                                                                  if amount == total:
                                                                        var = input(info + ' ')
                                                                        vardict[info] = var
                                                                        var = vardict.get(info)
                                                                        numberlist.append(amount)
                                                                        vardict.update()
                                                                        count = count + 1
                                                      elif num == 0:
                                                            if var == '?':
                                                                  while amount != total:
                                                                        print(clist[amount])
                                                                        amount = amount + 1
                                                                  if amount == total:
                                                                        var = input(info + ' ')
                                                                        vardict[info] = var
                                                                        var = vardict.get(info)
                                                                        numberlist.append(amount)
                                                                        vardict.update()
                                                                        count = count + 1                                                                              
                                                                                                                                   
                                                except AttributeError:
                                                      var = "input"
                                                      clist.append(var)
                                                      clist.append(count)
                                                      return var
                                                      
                                                
                        # Hash function
                        except AttributeError:
                              try:
                                    # Hash function
                                    var = re.search(r'\#',line)
                                    var = var.group(0)
                                    has = ''
                                    has += var
                                    try:
                                          # Hash variable
                                          var = re.search(r'to',line)
                                          var = var.group(0)
                                          hashs = ''
                                          hashs += var
                                    
                                          try:
                                                # what is going to be hashed
                                                var = re.search(r'\w+$',line)
                                                var = var.group(0)
                                                hashss = ''
                                                hashss += var
                                                hashss = hashss.replace(' ','')
                                                
                                                try:
                                                      # Replace Hash, Hash variable, what is being hashed
                                                      # The last variable
                                                      # This variable is going to be added
                                                      var = re.search(r'.+',line)
                                                      var = var.group(0)
                                                      var = var.replace(has, '')
                                                      var = var.replace(hashs, '')
                                                      var = var.replace(hashss, '')
                                                      var = var.replace(' ','')
                                                      newvar = ''
                                                      hashvar = ''
                                                      chvar = ''
                                                      try:
                                                            # Hash to a file
                                                            hashvar = vardict.get(hashss)
                                                            filez = ''
                                                            chvar = vardict.get(var)
                                                            filez = os.path.isfile(chvar)
                                                            if filez == True:
                                                                  file = open(chvar, 'a')
                                                                  
                                                                  file.write(hashvar + '\n')
                                                                  
                                                                  file.close()
                                                                  count = count + 1
                                                            else:
                                                                  newvar = chvar + hashvar
                                                                  vardict[var] = newvar
                                                                  vardict.update()
                                                                  count = count + 1
                                                      except TypeError:
                                                            vardict[hashss] = ''
                                                            vardict[var] = ''
                                                            vardict[newvar] = ''
                                                            vardict.update()
                                                            count = count + 1
                                                            

                                                       
                                                except AttributeError:
                                                      var = "Last Hash"
                                                      clist.append(count)
                                                      clist.append(var)
                                                      

                                          except AttributeError:
                                                var = "Hash var"
                                                clist.append(count)
                                                clist.append(var)
                                                
                                          
                                          
                                          
                                    except AttributeError:
                                          var = "to missing"
                                          clist.append(count)
                                          clist.append(var)
                                          
                              # writing files
                              except AttributeError:
                                    try:
                                          var = re.search(r'\[w]',line)
                                          var = var.group(0)
                                          try:
                                                # file to know what is being written
                                                var = re.search(r'file',line)
                                                var = var.group(0)
                                                filename = ''
                                                fileinput = ''
                                                num = 0
                                                num = len(numberlist)
                                                amount = 0
                                                total = 0
                                                total = len(clist)
                                                # Similar output function for input
                                                
                                                if num > 0:
                                                      numberlist.reverse()
                                                      amount = numberlist[0]
                                                      while amount != total:
                                                            print(clist[amount])
                                                            amount = amount + 1
                                                            if amount == total:
                                                                  filename = input()
                                                                  file = open(filename, 'w')
                                                                  fileinput = input()
                                                                  file.write(fileinput + '\n')
                                                                  file.close()
                                                                  numberlist.append(amount)
                                                                  count = count + 1
                                                                        
                                                elif num == 0:
                                                      while amount != total:
                                                            print(clist[amount])
                                                            amount = amount + 1
                                                            if amount == total:
                                                                  filename = input()
                                                                  file = open(filename, 'w')
                                                                  fileinput = input()
                                                                  file.write(fileinput + '\n')
                                                                  file.close()
                                                                  numberlist.append(amount)
                                                                  count = count + 1
                                                                        
                                                
                                          except AttributeError:
                                                var = "No file"
                                                clist.append(var)
                                                clist.append(count)
                                                
                                                                                                                                                      
                                    # reading files
                                    except AttributeError:
                                          try:
                                                var = re.search(r'\[r]',line)
                                                var = var.group(0)

                                                try:
                                                      # file to know what is being read
                                                      var = re.search(r'file',line)
                                                      var = var.group(0)
                                                      var = var.replace(' ','')
                                                      filename = ''
                                                      fileinput = ''
                                                      num = 0
                                                      num = len(numberlist)
                                                      amount = 0
                                                      total = 0
                                                      total = len(clist)
                                                      
                                                      if num > 0:
                                                            numberlist.reverse()
                                                            amount = numberlist[0]
                                                            while amount != total:
                                                                  print(clist[amount])
                                                                  amount = amount + 1
                                                                  if amount == total:
                                                                        try:
                                                                              filename = input()
                                                                              file = open(filename, 'r')

                                                                              for linz in file:
                                                                                    print(linz)
                                                                              
                                                                              file.close()
                                                                              numberlist.append(amount)
                                                                              count = count + 1
                                                                        except FileNotFoundError:
                                                                              var = "file not found"
                                                                              clist.append(count)
                                                                              clist.append(var)
                                                                              
                                                                        
                                                      elif num == 0:
                                                            while amount != total:
                                                                  print(clist[amount])
                                                                  amount = amount + 1
                                                                  if amount == total:
                                                                        try:
                                                                              filename = input()
                                                                              file = open(filename, 'r')

                                                                              for linz in file:
                                                                                    print(linz)

                                                                              file.close()      
                                                                              numberlist.append(amount)
                                                                              count = count + 1
                                                                        except FileNotFoundError:
                                                                              var = "file not found"
                                                                              clist.append(count)
                                                                              clist.append(var)
                                                                              
                                                except AttributeError:
                                                      var = "No file"
                                                      clist.append(count)
                                                      clist.append(var)
                                                            
                                                            
                                          # reading and writing files                                                                       
                                          except AttributeError:
                                                try:
                                                      var = re.search(r'\[rw]',line)
                                                      var = var.group(0)
      
                                                      try:
                                                            # file to know what is being read
                                                            var = re.search(r'file',line)
                                                            var = var.group(0)
                                                            var = var.replace(' ','')
                                                            filename = ''
                                                            fileinput = ''
                                                            num = 0
                                                            num = len(numberlist)
                                                            amount = 0
                                                            total = 0
                                                            total = len(clist)
                                                            
                                                            if num > 0:
                                                                  numberlist.reverse()
                                                                  amount = numberlist[0]
                                                                  while amount != total:
                                                                        print(clist[amount])
                                                                        amount = amount + 1
                                                                        if amount == total:
                                                                              try:
                                                                                    filename = input()
                                                                                    fileinput = ''
                                                                                    # Open file
                                                                                    file = open(filename, 'r')
      
                                                                                    for linz in file:
                                                                                          print(linz)
                                                                                    file.close()
                                                                                    # Add to file
                                                                                    file = open(filename, 'a')
                                                                                    fileinput = input()
                                                                                    file.write(fileinput + '\n')
                                                                                    file.close()
                                                                                    numberlist.append(amount)
                                                                                    count = count + 1
                                                                              except FileNotFoundError:
                                                                                     var = "file not found"
                                                                                     clist.append(count)
                                                                                     clist.append(var)
                                                                              
                                                                              
                                                            elif num == 0:
                                                                  while amount != total:
                                                                        print(clist[amount])
                                                                        amount = amount + 1
                                                                        if amount == total:
                                                                              try:
                                                                                    filename = input()
                                                                                    fileinput = ''
                                                                                    # Open file
                                                                                    file = open(filename, 'r')
      
                                                                                    for linz in file:
                                                                                          print(linz)
                                                                                    file.close()
                                                                                    # Add to file
                                                                                    file = open(filename, 'a')
                                                                                    fileinput = input()
                                                                                    file.write(fileinput + '\n')
                                                                                    file.close()
                                                                                    numberlist.append(amount)
                                                                                    count = count + 1
                                                                              except FileNotFoundError:
                                                                                     var = "file not found"
                                                                                     clist.append(count)
                                                                                     clist.append(var)
                                                                                    
                                                                              
                                                                              
                                                      except AttributeError:
                                                            var = "No file"
                                                            clist.append(count)
                                                            clist.append(var)
                                                            

                                                # Check test expressions
                                                # it will only return true or false statements
                                                except AttributeError:
                                                      try:
                                                            var = re.search(r'\[check]',line)
                                                            var = var.group(0)
                                                            try:
                                                                  var = re.search(r'\s.+',line)
                                                                  var = var.group(0)
                                                                  try:
                                                                        var = eval(var)
                                                                  except NameError:
                                                                        var = 'only numbers'
                                                                        clist.append(var)
                                                                        return var
                                                                  
                                                                  if var == True:
                                                                        clist.append(True)
                                                                        count = count + 1
                                                                  else:
                                                                        clist.append(False)
                                                                        count = count + 1
                                                                        
                                                                        
                                                                        
                                                            except AttributeError:
                                                                  var = "No file"
                                                                  clist.append(count)
                                                                  clist.append(var)
                                                      # Comments
                                                      # skips line when comming to comments
                                                      except AttributeError:
                                                            try:
                                                                  var = re.search(r'\//',line)
                                                                  var = var.group(0)
                                                                  count = count + 1
                                                                  
                                                            # Opening a web browser
                                                            except AttributeError:
                                                                  try:
                                                                        var = re.search(r'\[web]',line)
                                                                        var = var.group(0)
                                                                        try:
                                                                              web = ''
                                                                              # searches for website
                                                                              web = re.search(r'\s.+',line)
                                                                              web = web.group(0)
                                                                              web = web.replace(' ','')
                                                                              Header = 'http://'
                                                                              url = Header + web
                                                                              new=2;
                                                                              url= url;
                                                                              webbrowser.open(url,new=new)
                                                                              count = count + 1
                                                                        except AttributeError:
                                                                              var = "no website"
                                                                              clist.append(count)
                                                                              clist.append(var)
                                                                              
                                                            
                                                                  
                                                                  
                                                                  # exit function closes the Interpreter
                                                                  # depending where you put the [exit] function
                                                                  # may determine if output shows
                                                                  
                                                                  except AttributeError:
                                                                        try:
                                                                              var = re.search(r'\[exit]',line)
                                                                              var = var.group(0)
                                                                              break

                                                                        # move files 
                                                                        
                                                                        except AttributeError:
                                                                              try:
                                                                                    var = re.search(r'\[move]',line)
                                                                                    var = var.group(0)
                                                                                    try:
                                                                                          file = ''
                                                                                          location = ''
                                                                                          var = re.search(r'\s.+\s-',line)
                                                                                          var = var.group(0)
                                                                                          file += var
                                                                                          file = file.replace('-','')
                                                                                          file = file.replace(' ','')
                                                                                          line = line.replace('[move]','')
                                                                                          line = line.replace(var,'')
                                                                                          line = line.replace(' ','')
                                                                                          location += line
                                                                                          try:
                                                                                                
                                                                                                shutil.move(file,location)
                                                                                                count = count + 1
                                                                                          except FileNotFoundError:
                                                                                                var = 'File or directory not Found'
                                                                                                clist.append(count)
                                                                                                clist.append(var)
                                                                                                
                                                                                          
                                                                                          
                                                                                          
                                                                                    except AttributeError:
                                                                                          var = 'no file'
                                                                                          clist.append(count)
                                                                                          clist.append(var)
                                                                                          
                                                                              # copy files                                                                                           
                                                                              except AttributeError:
                                                                                    try:
                                                                                          var = re.search(r'\[copy]',line)
                                                                                          var = var.group(0)
                                                                                          
                                                                                          
                                                                                    
                                                                                          try:
                                                                                                file = ''
                                                                                                location = ''
                                                                                                var = re.search(r'\s.+\s-',line)
                                                                                                var = var.group(0)
                                                                                                file += var
                                                                                                file = file.replace('-','')
                                                                                                file = file.replace(' ','')
                                                                                                line = line.replace('[copy]','')
                                                                                                line = line.replace(var,'')
                                                                                                line = line.replace(' ','')
                                                                                                location += line
                                                                                                try:
                                                                                                
                                                                                                      shutil.copy(file,location)
                                                                                                      count = count + 1
                                                                                                except FileNotFoundError:
                                                                                                      var = 'File or directory not Found'
                                                                                                      clist.append(count)
                                                                                                      clist.append(var)
                                                                                                
                                                                                                
                                                                                          except AttributeError:
                                                                                                var = 'no file'
                                                                                                clist.append(count)
                                                                                                clist.append(var)
                                                                                                
                                                                                                
                                                                                                
                                                                                          
                                                                                    # remove files                                                                                                
                                                                                    except AttributeError:
                                                                                          try:
                                                                                                var = re.search(r'\[remove]',line)
                                                                                                var = var.group(0)
                                                                                    
                                                                                                try:
                                                                                                      file = ''
                                                                                                      var = re.search(r'\s.+$',line)
                                                                                                      var = var.group(0)
                                                                                                      file += var
                                                                                                      file = file.replace(' ','')
                                                                                                      try:
                                                                                                      
                                                                                                            os.remove(file)
                                                                                                            count = count + 1
                                                                                                      except FileNotFoundError:
                                                                                                            var = 'File or directory not Found'
                                                                                                            clist.append(count)
                                                                                                            clist.append(var)
                                                                                                      
                                                                                                      
                                                                                                except AttributeError:
                                                                                                      var = 'no file'
                                                                                                      clist.append(count)
                                                                                                      clist.append(var)
                                                                                                
                                                                                                
                                                                                                
                                                                                          # make a directory 
                                                                                          except AttributeError:
                                                                                                try:
                                                                                                      var = re.search(r'\[mkdir]',line)
                                                                                                      var = var.group(0)
                                                                                                      try:
                                                                                                            var = re.search(r'\s.+$',line)
                                                                                                            var = var.group(0)
                                                                                                            var = var.replace(' ','')
                                                                                                            try:
                                                                                                                  os.mkdir(var)
                                                                                                                  count = count + 1
                                                                                                            except (FileNotFoundError,FileExistsError) as error :
                                                                                                                  var = 'directory not found or it exists'
                                                                                                                  clist.append(count)
                                                                                                                  clist.append(var)
                                                                                                            
                                                                                                      except AttributeError:
                                                                                                            var = 'no directory'
                                                                                                            clist.append(count)
                                                                                                            clist.append(var)
                                                                                                            
                                                                                                # remove directory
                                                                                                except AttributeError:
                                                                                                      try:
                                                                                                            var = re.search(r'\[rmdir]',line)
                                                                                                            var = var.group(0)

                                                                                                            try:
                                                                                                                  var = re.search(r'\s.+$',line)
                                                                                                                  var = var.group(0)
                                                                                                                  var = var.replace(' ','')
                                                                                                                  
                                                                                                                  try:
                                                                                                                        os.rmdir(var)
                                                                                                                        count = count + 1
                                                                                                                  except FileNotFoundError:
                                                                                                                        
                                                                                                                        var = 'directory not found'
                                                                                                                        clist.append(count)
                                                                                                                        clist.append(var)
                                                                                                            
                                                                                                            except AttributeError:
                                                                                                                  var = 'no directory'
                                                                                                                  clist.append(count)
                                                                                                                  clist.append(var)
                                                                                                            
                                                                                                      # get function
                                                                                                      # gets the data from a file and assigns it to a variable
                                                                                                      
                                                                                                      except AttributeError:
                                                                                                            try:
                                                                                                                  var = re.search(r'\[get]',line)
                                                                                                                  var = var.group(0)
                                                                                                                  try:
                                                                                                                        var =  re.search(r'\s.+\s-',line)
                                                                                                                        var = var.group(0)
                                                                                                                        line = line.replace('[get]','')
                                                                                                                        line = line.replace(var,'')
                                                                                                                        var = var.replace('-', '')
                                                                                                                        var = var.replace(' ','')
                                                                                                                        filenam = ''
                                                                                                                        filenam += var
                                                                                                                        filevar = ''
                                                                                                                        line = line.replace('\n','')
                                                                                                                        line = line.replace(' ','')
                                                                                                                        filevar += line
                                                                                                                        try:
                                                                                                                              file = open(filenam, 'r')
                                                                                                                              infos = ''

                                                                                                                              for linz in file:
                                                                                                                                    
                                                                                                                                    infos += linz
                                                                                                                                    
                                                                                                                              file.close()
                                                                                                                              vardict[filevar] = infos
                                                                                                                              vardict.update()
                                                                                                                              count = count + 1
                                                                                                                        except FileNotFoundError:
                                                                                                                              var = 'no file'
                                                                                                                              clist.append(count)
                                                                                                                              clist.append(var)
                                                                                                                              

                                                                                                                  except AttributeError:
                                                                                                                        var = 'no line'
                                                                                                                        clist.append(count)
                                                                                                                        clist.append(var)
                                                                                                                        
                                                                                                            # takes two arguments
                                                                                                            # copies all files from a certain
                                                                                                            # directory to anoter
                                                                                                            except AttributeError:
                                                                                                                  try:
                                                                                                                        var = re.search(r'\[take]',line)
                                                                                                                        var = var.group(0)
                                                                                                                        try:
                                                                                                                              var = re.search(r'\s.+\s-',line)
                                                                                                                              var = var.group(0)
                                                                                                                              path = ''
                                                                                                                              path += var
                                                                                                                              path = path.replace('-','')
                                                                                                                              path = path.replace(' ','')
                                                                                                                              line = line.replace('[take]','')
                                                                                                                              line = line.replace(var,'')
                                                                                                                              line = line.replace(' ','')
                                                                                                                              line = line.replace('\n','')
                                                                                                                              
                                                                                                                              location = ''
                                                                                                                              location += line
                                                                                                                              oslist = []
                                                                                                                              oslist = os.listdir(path)
                                                                                                                              totalss = 0
                                                                                                                              counter = 0
                                                                                                                              totalss = len(oslist)
                                                                                                                              while counter != totalss:
                                                                                                                                    file = ''
                                                                                                                                    file = oslist[counter]
                                                                                                                                    osfiles = path + file
                                                                                                                                    if os.path.isfile(osfiles) == True:
                                                                                                                                          shutil.copy(osfiles,location)
                                                                                                                                          counter = counter + 1
                                                                                                                                    else:
                                                                                                                                          counter = counter + 1
                                                                                                                                          clist.append('completed')
                                                                                                                              
                                                                                                                                          
                                                                                                                                          
                                                                                                                        except AttributeError:
                                                                                                                              var = 'no line'
                                                                                                                              clist.append(count)
                                                                                                                              clist.append(var)
                                                                                                                              
                                                                                                                              
                                                                                                                              
                                                                                                                              
                                                                                                                  except AttributeError:                                                                                                                       
                                                                                                                        var = 'variable unknow'
                                                                                                                        clist.append(count)
                                                                                                                        clist.append(var)
                                                                                                                  

                                                                                                            
                                                                                                            
                                                                                                                                                                                                                                    
                                                                                                                        
                                                                              

                                                                              
                                                                              
                                                                              
                                                      
                                                            
                                                      
                                                                         
                                                      
                  count = count + 1
      #return var
      
      
      
                                    
            

def interpret(var):
      count = 0
      num = 0
      num = len(numberlist)
      if num != 0:
            numberlist.reverse()
            count = numberlist[0]
      else:
            count = 0
      
      
      length = len(clist)
      while count < length:
            line = clist[count]
            print(line)
            count = count + 1

      print('')
      #print(vardict)
      #print(" ")
      #print(numberlist)
      #print(clist)
      #print(" ")
      #print(varlist)
      #print(var)
      #print(" ")
      #print(len(varlist))

def Shell():
      command = ''
      command = input()
      while command != '[systemc]':
            if command == '':
                  command = input()
            try:  # Digits
                  var = re.search(r'\d.+', command)
                  var = var.group(0)
                  if var != "None":
                        var = eval(var)
                        print(var)
                        command = ''
                        
                        
            except AttributeError:
                  try:
                        var = re.search(r'\[w]',command)
                        var = var.group(0)
                        try:
                              # file to know what is being written
                              var = re.search(r'file',command)
                              var = var.group(0)
                              
                              filename = input()
                              file = open(filename, 'w')
                              fileinput = input()
                              file.write(fileinput + '\n')
                              file.close()
                              command = ''
                        except AttributeError:
                              var = 'file argument missing'
                              print(var)
                              command = ''
                              
                              
                              
                  except AttributeError:
                        try:
                              var = re.search(r'\[r]',command)
                              var = var.group(0)
                              try:
                                    # file to know what is being written
                                    var = re.search(r'file',command)
                                    var = var.group(0)
                                    try:
                                          
                                          filename = input()
                                          file = open(filename, 'r')

                                          for linz in file:
                                                print(linz)
                                          
                                          file.close()
                                          command = ''
                                    except  FileNotFoundError:
                                          var = 'file not found'
                                          print(var)
                                          command = ''
                              except AttributeError:
                                    var = 'file argument missing'
                                    print(var)
                                    command = ''
                              
                        except AttributeError:
                              try:
                                    var = re.search(r'\[rw]',command)
                                    var = var.group(0)
      
                                    try:
                                          # file to know what is being read
                                          var = re.search(r'file',command)
                                          var = var.group(0)
                                          var = var.replace(' ','')
                                          filename = ''
                                          fileinput = ''
                                          try:
                                                filename = input()
                                                file = open(filename, 'r')

                                                for linz in file:
                                                      print(linz)
                                          
                                                file.close()

                                                file = open(filename, 'w')
                                                fileinput = input()
                                                file.write(fileinput + '\n')
                                                file.close()
                                          except  FileNotFoundError:
                                                var = 'file not found'
                                                print(var)
                                                command = ''
                                          
                                    except AttributeError:
                                          var = 'file argument missing'
                                          print(var)
                                          command = ''
                                          
                              except AttributeError:
                                    var = 'nothing read'
                                    print(var)
                                    command = ''
                              
                            
                                                                        

                   
                   
      

def run():
      filename = ''
      filename = open_file(filename)
      var = info(filename)
      interpret(var)
run()

