

# Created by Genesis
# Turtle Programming language
# System Shell Habitat
# IDE for Turtle programming language

varlist = []
vardict = {}
clist = []
numberlist = []

from sys import *
import re
import sys
import os
import webbrowser
import shutil

import tkinter

from tkinter import Text, Tk

class system:
      def __init__(self):

            self.main_window = tkinter.Tk()

            self.topframe = tkinter.Frame()
            self.midframe = tkinter.Frame()
            self.bottomframe = tkinter.Frame()

            self.mainlabel = tkinter.Label(self.topframe, \
                                           text=' Turtle ')
            self.mainlabel2 = tkinter.Label(self.topframe, \
                                            text=' System Shell Habitat ')
            
            self.mainlabel.config(font=("Menlo",15))
            self.mainlabel2.config(font=("Menlo",14))

            self.mainlabel.pack(side='top')
            self.mainlabel2.pack(side='top')
            
            self.Filename = tkinter.Entry(self.midframe, \
                                          width=16)
            self.Filename.pack(side='top')
            
            self.Writer_entry = Text(self.main_window, height=30, width=60)
            self.Writer_entry.config(font=("Menlo",13))
            self.Writer_entry.pack(side='bottom')


            self.write_button = tkinter.Button(self.bottomframe, \
                                               text='write', \
                                               command = self.write)
            
            self.delete_button = tkinter.Button(self.bottomframe, \
                                                text='Delete', \
                                                command = self.delete)
            
            self.read_button = tkinter.Button(self.bottomframe, \
                                              text='read', \
                                              command = self.read)
            
            self.run_button = tkinter.Button(self.bottomframe, \
                                             text='run', \
                                             command = self.run)
            
            self.quit_button = tkinter.Button(self.bottomframe, \
                                              text='exit', \
                                              command = self.main_window.destroy)


            self.write_button.pack(side='left')
            self.read_button.pack(side='left')
            self.run_button.pack(side='left')
            self.quit_button.pack(side='left')
            self.delete_button.pack(side='left')

            self.topframe.pack()
            self.midframe.pack()
            self.bottomframe.pack()

            tkinter.mainloop()

      

      def run(self):
            
      def read(self):

            Filename = str(self.Filename.get())
            Creator = open(Filename, 'r')
            
            for line in Creator:
                  words = str(line)
                  self.Writer_entry.insert('1.0',words)
            Creator.close()
            
      def write(self):

            Filename = str(self.Filename.get())
            File = str(self.Writer_entry.get('1.0','end'))

            creator = open(Filename, 'w')
            creator.write('\n' + File + '\n') 
            creator.close()
            
      def delete(self):

            Filename = str(self.Filename.get())
            self.Writer_entry.delete('1.0','end')
            os.remove(Filename)
            
            
            



system()

            

            
