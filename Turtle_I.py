


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
      # filename that is going to be read by interpreter
      filename = ''
      filename = input()

      return filename

def info(filename): # file name passed into info function
      
      lister = " "
      count = 0
      length = 0
      line = ""
      var = ""
      con = 0
      

      # Opening file for reading
      
      try:
            file = open(filename,'r')

      # reading lines from the file
      # appending the lines to varlist
      
            while lister != "":
                  lister = file.readline()
                  varlist.append(lister)
      
            file.close()

      # If file does not read then this will happen
      
      except (TypeError,FileNotFoundError) as error:
            print('File not found')
            
            
                  
                  
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
                                                # Will evaluate digit or expressions
                                                var = var.replace(" .",'')
                                                var = eval(var)
                                                clist.append(var)
                                                try:
                                                      # look for period
                                                      var = re.search(r'\.', line)
                                                      var = var.group(0)
                                                      count = count + 1
                                                except AttributeError:
                                                      # No period 
                                                      clist.pop()
                                                      clist.append("NO PERIOD")
                                                      clist.append(count)
                                                      var = "NO PERIOD"
                                                      return var
                                                # if the variable cant be evaluated then append to list
                                          except (NameError,SyntaxError) as error:
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
                                          var = re.search(r'\s.+\b', line)
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
                                                      # to see if they have value
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
                                                                  # hash to variable
                                                                  newvar = chvar + hashvar
                                                                  vardict[var] = newvar
                                                                  vardict.update()
                                                                  count = count + 1
                                                      # if the variable are empty or any other
                                                      # problem set all variables equal to nothing
                                                      except (TypeError,SyntaxError) as error:
                                                            vardict[hashss] = ''
                                                            vardict[var] = ''
                                                            vardict[newvar] = ''
                                                            vardict.update()
                                                            count = count + 1
                                                            

                                                       
                                                except AttributeError:
                                                      var = "last Hash"
                                                      clist.append(count)
                                                      clist.append(var)
                                                      return var
                                                      
                                                      

                                          except AttributeError:
                                                var = "Hash var"
                                                clist.append(count)
                                                clist.append(var)
                                                return var
                                                
                                          
                                          
                                          
                                    except AttributeError:
                                          var = "to missing"
                                          clist.append(count)
                                          clist.append(var)
                                          return var
                                          
                              # writing files
                              except AttributeError:
                                    try:
                                          var = re.search(r'\[w]',line)
                                          var = var.group(0)
                                          try:
                                                # file to know what is being written
                                                var = re.search(r'\s.+$',line)
                                                var = var.group(0)
                                                var = var.replace(' ','')
                                                var = var.replace('\n','')
                                                clist.append(var)
                                                filename = ''
                                                filename += var
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
                                                                  file = open(filename, 'w')
                                                                  fileinput = input()
                                                                  file.write(fileinput + '\n')
                                                                  file.close()
                                                                  numberlist.append(amount)
                                                                  count = count + 1
                                                                        
                                                
                                          except AttributeError:
                                                var = "no file"
                                                clist.append(var)
                                                clist.append(count)
                                                return var
                                                
                                                                                                                                                      
                                    # reading files
                                    except AttributeError:
                                          try:
                                                var = re.search(r'\[r]',line)
                                                var = var.group(0)

                                                try:
                                                      # file to know what is being read
                                                      var = re.search(r'\s.+$',line)
                                                      var = var.group(0)
                                                      var = var.replace(' ','')
                                                      var = var.replace('\n','')
                                                      clist.append(var)
                                                      filename = ''
                                                      filename += var
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
                                                                              return var
                                                                              
                                                                        
                                                      elif num == 0:
                                                            while amount != total:
                                                                  print(clist[amount])
                                                                  amount = amount + 1
                                                                  if amount == total:
                                                                        try:
                                                                              
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
                                                                              return var
                                                                              
                                                except AttributeError:
                                                      var = "no file"
                                                      clist.append(count)
                                                      clist.append(var)
                                                      return var
                                                            
                                                            
                                          # reading and writing files                                                                       
                                          except AttributeError:
                                                try:
                                                      var = re.search(r'\[rw]',line)
                                                      var = var.group(0)
      
                                                      try:
                                                            # file to know what is being read
                                                            var = re.search(r'\s.+$',line)
                                                            var = var.group(0)
                                                            var = var.replace(' ','')
                                                            var = var.replace('\n','')
                                                            clist.append(var)
                                                            filename = ''
                                                            filename += var
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
                                                                                     return var
                                                                              
                                                                              
                                                            elif num == 0:
                                                                  while amount != total:
                                                                        print(clist[amount])
                                                                        amount = amount + 1
                                                                        if amount == total:
                                                                              try:
                                                                                    
                                                                                    
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
                                                                                     return var
                                                                                    
                                                                              
                                                                              
                                                      except AttributeError:
                                                            var = "no file"
                                                            clist.append(count)
                                                            clist.append(var)
                                                            return var
                                                                                                            
                                                # Check test expressions
                                                # it will only return true or false statements
                                                # It can only receive numbers 
                                                except AttributeError:
                                                      try:
                                                            var = re.search(r'\[check]',line)
                                                            var = var.group(0)
                                                            try:
                                                                  var = re.search(r'\s.+',line)
                                                                  var = var.group(0)
                                                                  varc = ''
                                                                  varc += var
                                                                  try:
                                                                        var = eval(var)
                                                                        
                                                                  except (NameError,SyntaxError) as error:
                                                                        var = 'only numbers'
                                                                        clist.append(var)
                                                                        return var
                                                                  
                                                                  if var == True:
                                                                        line = varc + ' True'
                                                                        clist.append(line)
                                                                        count = count + 1
                                                                  else:
                                                                        line = varc + ' False'
                                                                        clist.append(line)
                                                                        count = count + 1
                                                            
                                                                              

                                                                        
                                                            except AttributeError:
                                                                  var = "no file"
                                                                  clist.append(count)
                                                                  clist.append(var)
                                                                  return var
                                                      # Comments
                                                      # skips line when comming to comments
                                                      except AttributeError:
                                                            try:
                                                                  var = re.search(r'\//',line)
                                                                  var = var.group(0)
                                                                  count = count + 1
                                                                  
                                                            # Opening a web browser
                                                            # it does not need to have http://
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
                                                                        except (AttributeError, SyntaxError) as error:
                                                                              var = "no website"
                                                                              clist.append(count)
                                                                              clist.append(var)
                                                                              return var
                                                                              
                                                            
                                                                  
                                                                  
                                                                  # exit function closes the Interpreter
                                                                  # depending where you put the [exit] function
                                                                  # may determine if output shows
                                                                  
                                                                  except AttributeError:
                                                                        try:
                                                                              var = re.search(r'\[exit]',line)
                                                                              var = var.group(0)
                                                                              break

                                                                        # move files 
                                                                        # takes two arguments
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
                                                                                          line = line.replace('\n','')
                                                                                          location += line
                                                                                          try:
                                                                                                
                                                                                                shutil.move(file,location)
                                                                                                count = count + 1
                                                                                          except (FileNotFoundError,SyntaxError,OSError) as error:
                                                                                                var = 'file or directory not Found'
                                                                                                clist.append(count)
                                                                                                clist.append(var)
                                                                                                return var
                                                                                                
                                                                                          
                                                                                          
                                                                                          
                                                                                    except AttributeError:
                                                                                          var = 'no file'
                                                                                          clist.append(count)
                                                                                          clist.append(var)
                                                                                          return var
                                                                                          
                                                                              # copy files
                                                                              # same syntax for moving files
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
                                                                                                line = line.replace('\n','')
                                                                                                location += line
                                                                                                try:
                                                                                                
                                                                                                      shutil.copy(file,location)
                                                                                                      count = count + 1
                                                                                                except (FileNotFoundError,SyntaxError,OSError) as error:
                                                                                                      var = 'file or directory not Found'
                                                                                                      clist.append(count)
                                                                                                      clist.append(var)
                                                                                                      return var
                                                                                                
                                                                                                
                                                                                          except AttributeError:
                                                                                                var = 'no file'
                                                                                                clist.append(count)
                                                                                                clist.append(var)
                                                                                                return var
                                                                                                
                                                                                                
                                                                                                
                                                                                          
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
                                                                                                      file = file.replace('\n','')
                                                                                                      try:
                                                                                                            # syntax of this statement only require one argument
                                                                                                            os.remove(file)
                                                                                                            count = count + 1
                                                                                                      except (FileNotFoundError,SyntaxError,OSError) as error:
                                                                                                            var = 'file or directory not Found'
                                                                                                            clist.append(count)
                                                                                                            clist.append(var)
                                                                                                            return var
                                                                                                      
                                                                                                      
                                                                                                except AttributeError:
                                                                                                      var = 'no file'
                                                                                                      clist.append(count)
                                                                                                      clist.append(var)
                                                                                                      return var
                                                                                                
                                                                                                
                                                                                                
                                                                                          # make a directory 
                                                                                          except AttributeError:
                                                                                                try:
                                                                                                      var = re.search(r'\[mkdir]',line)
                                                                                                      var = var.group(0)
                                                                                                      try:
                                                                                                            var = re.search(r'\s.+$',line)
                                                                                                            var = var.group(0)
                                                                                                            var = var.replace(' ','')
                                                                                                            var = var.replace('\n','')
                                                                                                            try:
                                                                                                                  os.mkdir(var)
                                                                                                                  count = count + 1
                                                                                                            except (FileNotFoundError,FileExistsError,OSError) as error :
                                                                                                                  var = 'directory not found or it exists'
                                                                                                                  clist.append(count)
                                                                                                                  clist.append(var)
                                                                                                                  return var
                                                                                                            
                                                                                                      except AttributeError:
                                                                                                            var = 'no directory'
                                                                                                            clist.append(count)
                                                                                                            clist.append(var)
                                                                                                            return var
                                                                                                            
                                                                                                # remove directory
                                                                                                except AttributeError:
                                                                                                      try:
                                                                                                            var = re.search(r'\[rmdir]',line)
                                                                                                            var = var.group(0)

                                                                                                            try:
                                                                                                                  var = re.search(r'\s.+$',line)
                                                                                                                  var = var.group(0)
                                                                                                                  var = var.replace(' ','')
                                                                                                                  var = var.replace('\n','')
                                                                                                                  
                                                                                                                  
                                                                                                                  try:
                                                                                                                        shutil.rmtree(var)
                                                                                                                        count = count + 1
                                                                                                                  except (FileNotFoundError,SyntaxError,OSError) as error:
                                                                                                                        
                                                                                                                        var = 'directory not found'
                                                                                                                        clist.append(count)
                                                                                                                        clist.append(var)
                                                                                                                        return var
                                                                                                            
                                                                                                            except AttributeError:
                                                                                                                  var = 'no directory'
                                                                                                                  clist.append(count)
                                                                                                                  clist.append(var)
                                                                                                                  return var
                                                                                                            
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
                                                                                                                        except (FileNotFoundError,SyntaxError) as error:
                                                                                                                              var = 'no file'
                                                                                                                              clist.append(count)
                                                                                                                              clist.append(var)
                                                                                                                              return var
                                                                                                                              

                                                                                                                  except AttributeError:
                                                                                                                        var = 'no line'
                                                                                                                        clist.append(count)
                                                                                                                        clist.append(var)
                                                                                                                        return var
                                                                                                                        
                                                                                                            # takes two arguments
                                                                                                            # copies all files from a certain
                                                                                                            # directory to another directory
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
                                                                                                                        
                                                                                                                              
                                                                                                                                          
                                                                                                                                          
                                                                                                                        except (AttributeError,OSError) as error:
                                                                                                                              var = 'no line'
                                                                                                                              clist.append(count)
                                                                                                                              clist.append(var)
                                                                                                                              return var
                                                                                                                              
                                                                                                                              
                                                                                                                              
                                                                                                                  # output creates a file with the output of the program
                                                                                                                  
                                                                                                                  except AttributeError:
                                                                                                                        try:
                                                                                                                              var = re.search(r'\[output]',line)
                                                                                                                              var = var.group(0)
                                                                                                                              line = line.replace(var,'')
                                                                                                                              line = line.replace(' ','')
                                                                                                                              line = line.replace('\n','')
                                                                                                                              filename = ''
                                                                                                                              filename += line
                                                                                                                              filename = filename.replace('\n','')
                                                                                                                              
                                                                                                                              out = []

                                                                                                                              out += clist
                                                                                                                              out = str(out)
                                                                                                                              file = open(filename, 'w')

                                                                                                                              
                                                                                                                              file.write(out)
                                                                                                                                    


                                                                                                                              file.close()
                                                                                                                              
                                                                                                                        except AttributeError:
                                                                                                                              var = 'unknown variable'
                                                                                                                              clist.append(count)
                                                                                                                              clist.append(var)
                                                                                                                              return var
                                                                                                                              
                                                                                                            
                                                                                                            
                                                                                                                                                                                                                                    
                                                                                                                        
                                                                              

                                                                              
                                                                              
                                                                              
                                                      
                                                            
                                                      
                                                                         
                                                      
                  count = count + 1
      #return var
      
      
      
                                    
            

def interpret(var):
      count = 0
      num = 0
      num = len(numberlist)

      # code for showing output of program
      
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

      # Pause is for windows
      # So that program does not close so fast
      
      pause = ''
      pause = input()
      
      # These are for debugging purposes
      # You want to run them with the program to know what's going on
 
      
      #print(vardict)
      #print(" ")
      #print(numberlist)
      #print(clist)
      #print(" ")
      #print(varlist)
      #print(var)
      #print(" ")
      #print(len(varlist))


def run():
      # Run function holds the interpreter
      filename = ''
      filename = open_file(filename)            
      var = info(filename)
      interpret(var)
run()

