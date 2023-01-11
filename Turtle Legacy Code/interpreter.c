//go:build interpreter.go
/*
  ---- Nigele McCoy
 Developer and Creator of the turtle interpreter
 Creator of the turtle programming language
 
 Re-writing the interpreter from python to C
 C is portable
 The benefit is it's easier to use
 Main purpose of writing it in C is to make turtle
 an independent interpreter
 
 */


/*
 
 Interpreter works well
 Issuses:
 Cannot do anything recursively
 Limitations:
 Not Object Oriented
 
 
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>
#include <stdbool.h>
#include <math.h>

// Variable link list
// holds name of variable
// Holds value of variable
struct variable {
    char name[1000];
    char value[1000];
    struct variable *next;
};

typedef struct variable variables;
typedef variables *link;

link head = NULL;
link new = NULL;
link current = NULL;


struct content{
    char line[1000];
};

// If Structure
struct ifState{
    char ifLine[1000];
    struct content lines[1000];
    struct ifState *ifStates;
};

typedef struct ifState ifStat;
typedef  ifStat *ifSLine;

ifSLine headss = NULL;
ifSLine newss = NULL;
ifSLine currentss = NULL;

int ifCounter = -1;
int ifLineCount = 0;
bool ifTrue = false;
bool ifStruct = false;

// function link list
// function holds the name of the function
// it also has another struct that holds
// that has an array to hold the contents
// of the function
struct function{
    char funcName[1000];
    struct content lines[1000];
    struct function *newFunc;
};

typedef struct function func;
typedef func *chain;

chain heads = NULL;
chain news = NULL;
chain currents = NULL;

// loop
// looping structure
struct loop{
    char loopName[1000];
    struct content lines[1000];
    struct loop *newloop;
};

typedef struct loop loops;
typedef loops *looper;

looper headed = NULL;
looper newed = NULL;
looper currented = NULL;


int loopCount = -1;
int loopIncrement;
bool loopBool;



// function values to keep track
// of if we are gathering function
// contents
bool functionTrue = false;
int lineCounter = -1;

void interpreter(char *tokens);

int main(int argc, char** argv)
{
    // File
    FILE *file;
    // file name
    char *filename;
    // character limit
    char line[1000];
    // token to be executed
    char *token;
    //
    int variable_count = 0;
    int count = 0;
   
    while(1)
    {
        
        // Read the argument from the command line
        // Checking if the file is
        if(argv[1] != NULL)
        {
            // filename is commandline argument
            // Getting the file name of the file that will be interpreted
            filename = argv[1];
        }
        else
        {
            // REPL
            // Read Evaluate Print Loop
            // This means that when the interpreter recieves no
            // argument then run it in the repl mode
            // meaning it being ran as an interactive interpreter
           
            char showsplit[2] = " ";
            char *showtok;
            char *showtoks;
            char *loopToken;
            // Display everything after except show
            float shaddnum = 0;
            float shaddtotal = 0;
            float shsubnum = 0;
            float shsubtotal = 0;
            float shmultinum = 1;
            float shmultitotal = 0;
            float shdivnum = 1.0;
            float shdivtotal = 0;
            float sum_token = 0;
            float number_token = 0;
            bool add_token = false;
            bool sub_token = false;
            bool multi_token = false;
            bool div_token = false;
            int shnum0 = 0;
            int shnums1 = 0;
            size_t len;
            int showValue = 0;
            int addvalue = 0;
            int subvalue = 0;
            int multivalue = 0;
            int divvalue = 0;
            int modvalue = 0;
            int equal;
            char *string;
            char *place;
            int place_Comment = 0;
            int place_show = 0;
            int count = 0;
            char *command = (char *) malloc(1000);
            
            while (!strstr(command,"[exit]"))
            {
               
                fgets(command,1000,stdin);
                
                // Getting the function definition
                // syntax def function()
                if(strstr(command,"def") || functionTrue == true)
                {
                    if(lineCounter == -1)
                    {
                        news = (chain)malloc(sizeof(func));
                        news->newFunc = heads;
                        heads = news;
                        functionTrue = true;
                        lineCounter = 0;
                        strcpy(news->funcName,command);
                    }
                    else if(lineCounter == 0)
                    {
                        strcpy(news->lines[lineCounter].line,command);
                        lineCounter++;
                    }
                    else if(lineCounter > 0)
                    {
                        if(strstr(command,"[end]") > 0)
                        {
                            strcpy(news->lines[lineCounter].line,command);
                            currents = heads;
                            while(currents->newFunc != NULL)
                            {currents = currents->newFunc;}
                            lineCounter = -1;
                            functionTrue = false;
                        }
                        else
                        {
                            strcpy(news->lines[lineCounter].line,command);
                            lineCounter++;
                        }
                    }
                }
                else if(strstr(command,"()") > 0)
                {
                    int defcount = 0;
                    currents = heads;
                    while(currents != NULL)
                    {
                        if(strstr(currents->funcName,command) > 0)
                        {
                            while (!(strstr("[end]",currents->lines[defcount].line) > 0))
                            {
                                //puts("here");
                                 char *stringToken = (char *) malloc(1000);
                                strcpy(stringToken,currents->lines[defcount].line);
                                interpreter(stringToken);
                                //puts("here");
                                defcount++;
                                //free(stringToken);
                            }
                            break;
                        }
                        else
                        {
                            currents = currents->newFunc;
                        }
                    }
                }
                else if(strstr(command,"[if]") > 0 )
                {
                    // puts("here 0");
                    ifStruct = true;
                    newss = (ifSLine)malloc(sizeof(ifStat));
                    newss->ifStates = headss;
                    headss = newss;
                    ifCounter = 1;
                    strcpy(newss->ifLine,command);
                    strcpy(newss->lines[0].line,command);
                    
                }
                else if(ifCounter == 1)
                {
                    //puts("here 1");
                    strcpy(newss->lines[ifCounter].line,command);
                    ifCounter++;
                    
                }
                else if(ifCounter > 0)
                {
                    // variable to check to make sure structure statment
                    // is true
                    
                    //fgets(command,1000,stdin);
                    //puts("here 2");
                    if(strstr(command,"[end]") > 0 || strstr(command,"[end]\n") > 0)
                    {
                        
                        //puts("here 3");
                        strcpy(newss->lines[ifCounter].line,command);
                        currentss = headss;
                        while(currentss->ifStates != NULL)
                        {currentss = currentss->ifStates;}
                        ifCounter = -1;
                        functionTrue = false;
                        
                        currentss = headss;
                        while(currentss != NULL)
                        {
                            
                            if(ifTrue == false)
                            {
                                while(!(strstr("[end]",currentss->lines[ifLineCount].line) > 0))
                                {
                                    
                                    char *ConditionToken = (char *) malloc(1000);
                                    strcpy(ConditionToken,currentss->lines[ifLineCount].line);
                                    
                                    if(strstr(ConditionToken,"[if]") > 0)
                                    {
                                        float floValue1;
                                        float floValue2;
                                        char  *strValue1;
                                        char  *strValue2;
                                        bool boValue1;
                                        bool boValue2;
                                        bool state;
                                        bool valueState =false;
                                        int which = -1;
                                        bool and,or,not =false;
                                        bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                        char *ifCondition = (char *) malloc(1000);
                                        ifCondition = strtok(ConditionToken," ");
                                        while(ifCondition != NULL)
                                        {
                                            //printf("%s ",ifCondition);
                                            if(strstr(ifCondition,"[if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"and") > 0)
                                                and = true;
                                            else if(strstr(ifCondition,"or") > 0)
                                                or = true;
                                            else if(strstr(ifCondition,"not") > 0)
                                                not = true;
                                            else if(strstr(ifCondition,"==") > 0)
                                                equal = true;
                                            else if(strstr(ifCondition,">") > 0)
                                                greater = true;
                                            else if(strstr(ifCondition,"<") > 0)
                                                less = true;
                                            else if(strstr(ifCondition,">=") > 0)
                                                greatEqual = true;
                                            else if(strstr(ifCondition,"<=") > 0)
                                                lessEqual = true;
                                            else if(strstr(ifCondition,"!=") > 0)
                                                notEqual = true;
                                            else
                                            {
                                                if(isdigit(ifCondition[0]) && valueState == false)
                                                {
                                                    which = 0;
                                                    floValue1 = atof(ifCondition);
                                                    valueState = true;
                                                }
                                                else if(isdigit(ifCondition[0]) && valueState == true)
                                                {
                                                    floValue2 = atof(ifCondition);
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                                {       which = 1;
                                                    strcpy(strValue1,ifCondition);
                                                    valueState = true;
                                                    
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                                {
                                                    strcpy(strValue2,ifCondition);
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                                {
                                                    which = 2;
                                                    boValue1 = true;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                                {
                                                    boValue1 = false;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                                {
                                                    boValue2 = true;
                                                    valueState = false;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                                {
                                                    boValue2 = false;
                                                    valueState = false;
                                                }
                                                else if(valueState == false)
                                                {
                                                    
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue1 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = false;
                                                                    break;
                                                                }
                                                                
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue1[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                                else if(valueState == true)
                                                {
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue2 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = false;
                                                                    break;
                                                                }
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue2[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                            }
                                            
                                            
                                            ifCondition = strtok(NULL," ");
                                        }
                                        
                                        if(which == 0)
                                        {
                                            if(equal == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 > floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 < floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greatEqual == true)
                                            {
                                                if(floValue1 >= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(lessEqual == true)
                                            {
                                                if(floValue1 <= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(notEqual == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 != floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            
                                        }
                                        else if(which == 2)
                                        {
                                            if(and == true)
                                            {
                                                if(boValue1 && boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(or == true)
                                            {
                                                if(boValue1 || boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(not == true)
                                            {
                                                if(!boValue1)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(equal == true)
                                            {
                                                if(boValue1 == boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(less == true)
                                            {
                                                if(boValue1 < boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greater == true)
                                            {
                                                if(boValue1 > boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                        }
                                        else if(which == 1)
                                        {
                                            if(equal == true)
                                            {
                                                if(strValue1 == strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(strValue1 > strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(strValue1 < strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            
                                        }
                                        
                                        if(ifTrue == true)
                                        {
                                            //puts("here");
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                            {
                                                char *ConditionState = (char *) malloc(1000);
                                                strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                                interpreter(ConditionState);
                                                ifLineCount++;
                                            }
                                            //ifTrue = false;
                                            //break;
                                        }
                                        else
                                        {
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                                ifLineCount++;
                                            }
                                            ifTrue = false;
                                        }
                                    }
                                    else if(strstr(ConditionToken,"[then if]") > 0)
                                    {
                                        float floValue1;
                                        float floValue2;
                                        char  *strValue1;
                                        char  *strValue2;
                                        bool boValue1;
                                        bool boValue2;
                                        bool state;
                                        bool valueState =false;
                                        int which = -1;
                                        bool and,or,not =false;
                                        bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                        char *ifCondition = (char *) malloc(1000);
                                        
                                        ifCondition = strtok(ConditionToken," ");
                                        
                                        while(ifCondition != NULL)
                                        {
                                            //printf("%s ",ifCondition);
                                            if(strstr(ifCondition,"[if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"[then") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"and") > 0)
                                                and = true;
                                            else if(strstr(ifCondition,"or") > 0)
                                                or = true;
                                            else if(strstr(ifCondition,"not") > 0)
                                                not = true;
                                            else if(strstr(ifCondition,"==") > 0)
                                                equal = true;
                                            else if(strstr(ifCondition,">") > 0)
                                                greater = true;
                                            else if(strstr(ifCondition,"<") > 0)
                                                less = true;
                                            else if(strstr(ifCondition,">=") > 0)
                                                greatEqual = true;
                                            else if(strstr(ifCondition,"<=") > 0)
                                                lessEqual = true;
                                            else if(strstr(ifCondition,"!=") > 0)
                                                notEqual = true;
                                            else
                                            {
                                                if(isdigit(ifCondition[0]) && valueState == false)
                                                {
                                                    which = 0;
                                                    floValue1 = atof(ifCondition);
                                                    valueState = true;
                                                }
                                                else if(isdigit(ifCondition[0]) && valueState == true)
                                                {
                                                    floValue2 = atof(ifCondition);
                                                    valueState = false;
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                                {       which = 1;
                                                    strcpy(strValue1,ifCondition);
                                                    valueState = true;
                                                    
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                                {
                                                    strcpy(strValue2,ifCondition);
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                                {
                                                    which = 2;
                                                    boValue1 = true;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                                {
                                                    boValue1 = false;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                                {
                                                    boValue2 = true;
                                                    valueState = false;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                                {
                                                    boValue2 = false;
                                                    valueState = false;
                                                }
                                                else if(valueState == false)
                                                {
                                                    
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue1 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = false;
                                                                    break;
                                                                }
                                                                
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        //printf("%c",current->value[count]);
                                                                        strValue1[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                                else if(valueState == true)
                                                {
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue2 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = false;
                                                                    break;
                                                                }
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue2[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                            }
                                            
                                            
                                            ifCondition = strtok(NULL," ");
                                        }
                                        
                                        if(which == 0)
                                        {
                                            
                                            if(equal == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 > floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 < floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greatEqual == true)
                                            {
                                                if(floValue1 >= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(lessEqual == true)
                                            {
                                                if(floValue1 <= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(notEqual == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 != floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                        }
                                        else if(which == 2)
                                        {
                                            if(and == true)
                                            {
                                                if(boValue1 && boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(or == true)
                                            {
                                                if(boValue1 || boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(not == true)
                                            {
                                                if(!boValue1)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(equal == true)
                                            {
                                                if(boValue1 == boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(less == true)
                                            {
                                                if(boValue1 < boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greater == true)
                                            {
                                                if(boValue1 > boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                        }
                                        else if(which == 1)
                                        {
                                            if(equal == true)
                                            {
                                                if(strValue1 == strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(strValue1 > strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(strValue1 < strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            
                                        }
                                        
                                        if(ifTrue == true)
                                        {
                                            //puts("here");
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                            {
                                                //printf("%s",currentss->lines[ifLineCount].line);
                                                char *ConditionStates = (char *) malloc(1000);
                                                strcpy(ConditionStates,currentss->lines[ifLineCount].line);
                                                interpreter(ConditionStates);
                                                ifLineCount++;
                                                //puts("here");
                                            }
                                            //ifTrue = false;
                                            //break;
                                            
                                        }
                                        else
                                        {
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                                // printf("%s",currentss->lines[ifLineCount].line);
                                                //interpreter(currentss->lines[ifLineCount].line);
                                                ifLineCount++;
                                                
                                            }
                                            ifTrue = false;
                                        }
                                    }
                                    
                                    else if(strstr(ConditionToken,"[then]") > 0)
                                    {
                                        
                                        ifTrue = true;
                                        
                                        
                                        if(ifTrue == true)
                                        {
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[end]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[ended]") > 0)) {
                                                char *ConditionState = (char *) malloc(1000);
                                                strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                                interpreter(ConditionState);
                                                ifLineCount++;
                                                
                                            }
                                            //ifTrue = false;
                                            break;
                                            
                                        }
                                        else
                                            ifTrue = true;
                                        
                                    }
                                    if(ifTrue == true)
                                        break;
                                    else
                                        continue;
                                    
                                    
                                    
                                }
                            }
                            else
                                break;
                            
                        }
                    }
                    else
                    {
                        
                        //puts("here added");
                        strcpy(newss->lines[ifCounter].line,command);
                        ifCounter++;
                    }
                }
                 // Print data
                /*else if(strstr(command,"[loop]") >0)
                {
                    loopBool = true;
                    newed = (looper)malloc(sizeof(loops));
                    newed->newloop = headed;
                    headed = newed;
                    loopCount = 1;
                    strcpy(newed->loopName,command);
                   
                    
                    //puts("here loop");
                    loopToken = strtok(command," ");
                    while(loopToken != NULL)
                    {
                        if(isdigit(loopToken[0]))
                            loopIncrement = atoi(loopToken);
                        
                        loopToken = strtok(NULL," ");
                    }
                    
                    
                }
                else if(loopCount == 1)
                {
                   // puts("here added first loop");
                    strcpy(newed->lines[loopCount].line,command);
                    loopCount++;
                }
                else if(loopCount > 0)
                {
                    // puts("here added more than one");
                    if(strstr(command,"[end]") > 0 || strstr(command,"[end]\n") > 0)
                    {
                        strcpy(newed->lines[loopCount].line,command);
                        currented = headed;
                        while(currented->newloop != NULL)
                        {currented = currented->newloop;}
                        
                        int counters = 0;
                        count = 0;
                        while (count < loopIncrement)
                        {
                            counters = 0;
                            char *stringToken = (char *) malloc(1000);
                            while(counters < loopCount)
                            {
                                //puts("here added more than one");
                                
                                strcpy(stringToken,currented->lines[counters].line);
                                //printf("%s",stringToken);
                                interpreter(stringToken);
                                counters++;
                            }
                            count++;
                        }
                        
                        loopCount = -1;
                        count = 0;
                        
                    }
                    else
                    {
                        strcpy(newed->lines[loopCount].line,command);
                        loopCount++;
                        
                    }
                }*/
                else if(strstr(command,"show") > 0)
                {
                    // Interpreter is case sensitive
                    
                    // show function
                    // display words and numbers
                    // displays simple math expression
                    // syntax for show
                    // show hello
                    // show 1 + 3
                   
                    // Will do math if it finds an expression
                    string = strstr(command,"//");
                    place = strstr(command,"show");
                    place_Comment = string - command;
                    place_show = place - command;
                    // Dealing with comments
                    if(place_Comment < place_show)
                        continue;
                    else{
                        
                    if(strstr(command,"\"") > 0)
                    {
                        //showtok = strtok(token,"\"");
                        showtok = strtok(command,"\"");
                        while (showtok != NULL) {
                            
                            //printf("%s \n",showtok);
                            if (strstr(showtok,"show") > 0)
                                showValue = 1;
                            else
                            {
                                if (strstr(command,"\""))
                                {
                                    char *stringArray = (char *) malloc(1000);
                                    int count = 0;
                                    while(showtok[count] != '\"')
                                    {
                                        stringArray[count] = showtok[count];
                                        count++;
                                    }
                                    printf("%s \n",stringArray);
                                    free(stringArray);
                                }
                                else
                                {
                                    char *stringArray = (char *) malloc(1000);
                                    int count =0;
                                    while(showtok[count] != '\"')
                                    {
                                        stringArray[count] = showtok[count];
                                        count++;
                                    }
                                    printf("%s \n",stringArray);
                                    free(stringArray);
                                }
                            }
                            showtok = strtok(NULL,showsplit);
                        }
                        //printf("\n");
                    }
                    else if(strstr(command,"+") > 0  || strstr(command,"-") > 0 || strstr(command,"*") > 0 || strstr(command,"/") > 0)
                    {
                        showtok = strtok(command," ");
                        while (showtok != NULL) {
                            
                            if (strstr(showtok,"show") > 0)
                                showValue = 1;
                            else if (strstr(showtok,"+") > 0)
                                add_token = true;
                            else if (strstr(showtok,"-") > 0)
                                sub_token = true;
                            else if (strstr(showtok,"*") > 0)
                                multi_token = true;
                            else if (strstr(showtok,"/") > 0)
                                div_token = true;
                            else if(isnumber(showtok[0]))
                            {
                                if (number_token == 0)
                                {
                                    if (multi_token == true)
                                    {
                                        number_token = atof(showtok);
                                        sum_token = 1;
                                    }
                                    else if(div_token == true)
                                    {
                                        number_token = atof(showtok);
                                        sum_token = 1;
                                    }
                                    else
                                        number_token = atof(showtok);
                                }
                                else
                                    sum_token = atof(showtok);
                                
                                if(add_token == true)
                                {
                                    number_token += sum_token;
                                    add_token = false;
                                }
                                else if(sub_token == true)
                                { number_token -= sum_token;
                                    sub_token = false;
                                }
                                else if(multi_token == true)
                                {
                                    number_token *= sum_token;
                                    multi_token = false;
                                }
                                else if(div_token == true)
                                {   number_token /= sum_token;
                                    div_token = false;
                                }
                            }
                            showtok = strtok(NULL,showsplit);
                        }
                        // check for decimal place
                        // add that
                        printf("%3.3f \n",number_token);
                        number_token = 0;
                        sum_token = 0;
                        
                    }
                    else if(strstr(command,"%") > 0)
                    {
                        
                        showtok = strtok(command,showsplit);
                        while (showtok != NULL) {
                            
                            if (strstr(showtok,"%") > 0)
                                //printf("%s \n",showtok );
                                modvalue = 1;
                            else if (strstr(showtok,"\n") > 0)
                            {
                                //printf("%s \n",showtok );
                                modvalue = 1;
                            }
                            else if (shnum0 == 0)
                                shnum0 = ceil(atoi(showtok));
                            else if (shnum0 != 0)
                            {shnums1 = ceil(atoi(showtok));}
                            showtok = strtok(NULL,showsplit);
                        }
                        //shnum0 = ceil(shnum0);
                        //shnums1 = ceil(shnums1);
                        //shnum0 %= shnums1;
                        printf("%d \n",shnum0);
                    }
                    else
                    {
                       
                        showtok = strtok(command,showsplit);
                        while (showtok != NULL) {
                            
                            if (strstr(showtok,"show") > 0)
                            {
                                showValue = 1;
                            }
                            else
                            {
                                current = head;
                                while(current != NULL)
                                {
                                    if(strstr(showtok,current->name) > 0)
                                    {
                                        if (isdigit(current->value[0]))
                                        {
                                            printf("%s \n",current->value);
                                            break;
                                        }
                                        else
                                        {
                                            int size = 0;
                                            int count = 2;
                                            size = strlen(current->value);
                                            while (count <= size-3) {
                                                printf("%c",current->value[count]);
                                                count++;
                                            }
                                            //printf("%s",current->value);
                                            puts("");
                                            break;
                                        }
                                    }
                                    else
                                        current = current->next;
                                }
                            }
                            showtok = strtok(NULL,showsplit);
                        }
                    }
                    }
                }
                else if(strstr(command,"sys") > 0)
                {
                    
                    int sys_value = 0;
                    char *sys_token = (char *) malloc(1000);
                    showtok = strtok(command," ");
                    while (showtok != NULL) {
                        
                        if (strstr(showtok, "sys") > 0)
                            sys_value = 1;
                        else
                        {
                            strcat(sys_token,showtok);
                             strcat(sys_token," ");
                            //printf("%s ",sys_token);
                        }
                        showtok = strtok(NULL,showsplit);
                    }
                    system(sys_token);
                }
                else if(strstr(command,"[r]") > 0)
                {
                    FILE *readFile;
                    char *readingFile  = (char *) malloc(1000);
                    int read_value = 0;
                    char *read_token;
                    char *readLine  = (char *) malloc(1000);
                    char *lineRead  = (char *) malloc(1000);
                    
                    read_token = strtok(command," ");
                    while(read_token != NULL)
                    {
                        if (strstr(read_token,"[r]") > 0)
                        {
                            read_value = 1;
                           // printf("%s token \n",read_token);
                        }
                        else
                        {
                            int count = 0;
                            while (read_token[count] != '\n')
                            {
                                
                                readingFile[count] = read_token[count];
                                count++;
                            }
                            //  printf("%s file name \n",readingFile);
                            
                            if( (readFile = fopen(readingFile, "r+")) != NULL)
                            {
                                while(!feof(readFile))
                                {
                                    readLine =fgets(lineRead,1000,readFile);
                                    if (readLine == NULL)
                                        continue;
                                    else
                                        printf("%s",readLine);
                                }
                            }
                            else if( (readFile = fopen(readingFile, "r+")) == NULL)
                            {
                                printf("%s not found",readingFile);
                            }
                            
                        }
                        read_token = strtok(NULL," ");
                    }
                    puts("");
                    
                }
                else if(strstr(command,"[w]") > 0)
                {
                    FILE  *writeFile;
                    char *writingFile = (char *) malloc(1000);
                    int write_Value = 0;
                    char *write_token;
                    char *writeLine = (char *) malloc(1000);;
                    char *delimator = "\n";
                    write_token = strtok(command," ");
                    
                    while(write_token != NULL)
                    {
                        if(strstr(write_token,"[w]") > 0)
                            write_Value = 1;
                        else
                        {
                            //writingFile = write_token;
                            
                            int count = 0;
                            while (write_token[count] != '\n')
                            {
                                
                                writingFile[count] = write_token[count];
                                count++;
                            }
                            if( (writeFile  = fopen(writingFile, "w+")) != NULL )
                            {
                                while (!strstr(writeLine,delimator)) {
                                    
                                    fprintf(writeFile,"%s" ,fgets(writeLine,1000,stdin));
                                }
                                
                                fclose(writeFile);
                            }
                            else if ((writeFile  = fopen(writingFile, "w+")) == NULL)
                            {
                                printf("%s not found",writingFile);
                            }
                            
                            
                        }
                        
                        write_token = strtok(NULL," ");
                    }
                }
                else if(strstr(command,"[a]") > 0)
                {
                    FILE  *appendFile;
                    char *appendingFile = (char *) malloc(1000);
                    int append_Value = 0;
                    char *append_token;
                    char *appendLine = (char *) malloc(1000);
                    char *delimator = "\n";
                    append_token = strtok(command," ");
                    
                    while(append_token != NULL)
                    {
                        if(strstr(append_token,"[a]") > 0)
                            append_Value = 1;
                        else
                        {
                           
                            int count = 0;
                            while (append_token[count] != '\n')
                            {
                                
                                appendingFile[count] = append_token[count];
                                count++;
                            }
                            if( (appendFile  = fopen(appendingFile, "a+")) != NULL )
                            {
                                while (!strstr(appendLine,delimator)) {
                                    
                                    fprintf(appendFile,"%s" ,fgets(appendLine,1000,stdin));
                                }
                                
                                fclose(appendFile);
                            }
                            else if ((appendFile  = fopen(appendingFile, "a+")) == NULL)
                            {
                                printf("%s not found",appendingFile);
                            }
                            
                            
                        }
                        
                        append_token = strtok(NULL," ");
                    }
                    
                }
                else if (strstr(command,"[math]") > 0)
                {
                    // math token variables
                    // Math function just
                    // does simple math
                    // syntax [math] 1 + 3
                    char *math;
                    char *str;
                    // counter and len variable
                    size_t count = 0;
                    size_t len;
                    // math variables
                    float addnum = 0;
                    float addtotal = 0;
                    float subnum = 0;
                    float subtotal = 0;
                    float multinum = 0;
                    float multitotal = 0;
                    float divnum = 0;
                    float divtotal = 0;
                    int num0 = 0;
                    int nums1 = 0;
                    char *tokens;
                    char tok[2] = " ";
                    math =command;
                    
                    if(strstr(command,"+") > 0)
                    {
                        // function will loop and get the numbers out of the char array
                        // then run math operations on numbers
                       // adding
                        tokens = strtok(math,tok);
                        
                        while (tokens != NULL) {
                            
                                if (addnum == 0)
                                 addnum = atof(tokens);
                                 else if (addnum > 0)
                                 addtotal = atof(tokens);
                                tokens = strtok(NULL,tok);
                           
                        }
                        addtotal += addnum;
                        printf("%3.3f \n",addtotal);
                        
                    }
                    else if(strstr(command,"-") > 0)
                    {
                
                        // subtraction
                        tokens = strtok(math,tok);
                        
                        while (tokens != NULL) {
                           
                            
                            if (subtotal == 0)
                            subtotal = atof(tokens);
                            else if (subtotal > 0)
                            {subnum = atof(tokens);}
                            tokens = strtok(NULL,tok);
                            
                        }
                        subtotal -= subnum;
                        printf("%3.3f \n",subtotal);
                        
                    }
                    else if(strstr(command,"*") > 0)
                    {
                        // multiplication
                        tokens = strtok(math,tok);
                        
                        while (tokens != NULL) {
                            
                            if (multitotal == 0)
                            multitotal = atof(tokens);
                            else if (multitotal > 0)
                            {multinum = atof(tokens);}
                            tokens = strtok(NULL,tok);
                            
                        }
                        multitotal *= multinum;
                        printf("%3.3f \n",multitotal);
                        
                        
                    }
                    else if(strstr(command,"/") > 0)
                    {
                        
                        // Division
                        tokens = strtok(math,tok);
                        
                        while (tokens != NULL) {
                           
                            if (divtotal == 0)
                            divtotal = atof(tokens);
                            else if (divtotal > 0)
                            {divnum = atof(tokens);}
                            tokens = strtok(NULL,tok);
                            
                        }
                        divtotal /= divnum;
                        printf("%3.3f \n",divtotal);
                        
                    }
                    else if(strstr(command,"%") > 0)
                    {
                       // modulus
                        tokens = strtok(math,tok);
                        
                        while (tokens != NULL) {
                            
                            if (num0 == 0)
                            num0 = atof(tokens);
                            else if (num0 > 0)
                            {nums1 = atof(tokens);}
                            tokens = strtok(NULL,tok);
                            
                        }
                        num0 %= nums1;
                        printf("%d \n",num0);
                        
                    }
                    else
                        // if something is missing from show
                        // or math function
                        // then prints this statement
                        puts("operator missing");
                    
                    
                    
                }
                else if (strstr(command,"clear") > 0)
                {
                    // Clear screen
                    system(command);
                }
                else if (strstr(command,"[mkdir]") > 0)
                {
                    char *mkdir = (char *) malloc(1000);
                    char *mkdir_token;
                    int mkdir_value = 0;
                    
                    mkdir_token = strtok(command," ");
                    while (mkdir_token != NULL)
                    {
                        if(strstr(mkdir_token,"[mkdir]") > 0)
                            mkdir_value = 1;
                        else
                        {
                            strcat(mkdir,"mkdir ");
                            strcat(mkdir,mkdir_token);
                        }
                        mkdir_token = strtok(NULL," ");
                        
                    }
                    system(mkdir);
                    
                }
                else if(strstr(command,"[remove]") > 0)
                {
                    char *rmdir = (char *) malloc(1000);
                    char *rmdir_token;
                    int rmdir_value = 0;
                    
                    rmdir_token = strtok(command," ");
                    while (rmdir_token != NULL)
                    {
                        if(strstr(rmdir_token,"[remove]") > 0)
                            rmdir_value = 1;
                        else
                        {
                            strcat(rmdir,"rmdir ");
                            strcat(rmdir,rmdir_token);
                        }
                        rmdir_token = strtok(NULL," ");
                    }
                    
                    system(rmdir);
                }
                else if(strstr(command,"[copy]") > 0)
                {
                    // copy keyword is for windows
                    // find command for windows
                    
                    char *cp = (char *) malloc(1000);
                    char *copy_token = (char *) malloc(1000);
                    char *destName = (char *) malloc(1000);
                    char *newDestname = (char *) malloc(1000);
                    char *dash = (char *) malloc(1000);
                    int copy_value = 0;
                    int returnOld;
                    int returnNew;
                    FILE *old,*new;
                    int ch;
                    int count = 0;
                    char *new_token;
                    
                    new_token = command;
                    while (new_token[count] != ' ')
                    {
                        copy_token[count] = new_token[count];
                        count++;
                    }
                    
                    count++;
                    int newCount = 0;
                    
                    while (new_token[count] != ' ')
                    {
                        destName[newCount] = new_token[count];
                        count++;
                        newCount++;
                    }
                    
                    newCount = 0;
                    
                    while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
                    {
                        dash[newCount] = new_token[count];
                        count++;
                        newCount++;
                    }
                    newCount = 0;
                    
                    while (new_token[count] != '\n')
                    {
                        newDestname[newCount] = new_token[count];
                        
                        count++;
                        newCount++;
                    }
                    
                    if ((old = fopen(destName,"rb")) == NULL )
                        returnOld = -1;
                    
                    if ((new = fopen(newDestname,"wb")) == NULL )
                    {
                        fclose(old);
                        returnNew = -1;
                    }
                    
                    if (returnOld == -1)
                    {
                        printf("%s not found \n",destName);
                        return 0;
                    }
                    else if (returnNew == -1)
                    {
                        printf("%s not found \n",newDestname);
                        return 0;
                    }
                    else
                    {
                        while(1)
                        {
                            ch = fgetc(old);
                            
                            if(!feof(old))
                                fputc(ch,new);
                            else
                                break;
                        }
                        
                        fclose(new);
                        fclose(old);
                    }
                    
                }
                else if (strstr(command,"[move]") > 0)
                {
                    
                    char *copy_token = (char *) malloc(1000);
                    char *destName = (char *) malloc(1000);
                    char *newDestname = (char *) malloc(1000);
                    char *dash = (char *) malloc(1000);
                    int copy_value = 0;
                    int returnOld;
                    int returnNew;
                    FILE *old,*new;
                    int ch;
                    int count = 0;
                    char *new_token;
                    
                    
                    new_token = command;
                    while (new_token[count] != ' ')
                    {
                        copy_token[count] = new_token[count];
                        count++;
                    }
                    
                    count++;
                    int newCount = 0;
                    
                    while (new_token[count] != ' ')
                    {
                        destName[newCount] = new_token[count];
                        count++;
                        newCount++;
                    }
                    newCount = 0;
                    
                    while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
                    {
                        dash[newCount] = new_token[count];
                        count++;
                        newCount++;
                    }
                    newCount = 0;
                    
                    while (new_token[count] != '\n')
                    {
                        newDestname[newCount] = new_token[count];
                        count++;
                        newCount++;
                    }
                    
                    if ((old = fopen(destName,"rb")) == NULL )
                        returnOld = -1;
                    
                    if ((new = fopen(newDestname,"wb")) == NULL )
                    {
                        fclose(old);
                        returnNew = -1;
                    }
                    
                    if (returnOld == -1)
                    {
                        printf("%s not found \n",destName);
                        return 0;
                    }
                    else if (returnNew == -1)
                    {
                        printf("%s not found \n",newDestname);
                        return 0;
                    }
                    else
                    {
                        while(1)
                        {
                            ch = fgetc(old);
                            
                            if(!feof(old))
                                fputc(ch,new);
                            else
                                break;
                            
                        }
                        
                        fclose(new);
                        fclose(old);
                    }
                    
                    remove(destName);
                    
                }
                else if(strstr(command,"+=") > 0)
                {
                    char *addTok;
                    int addV = 0;
                    float addval = 0;
                    char *addVar = (char *) malloc(1000);
                    char *stringVal = (char *) malloc(1000);
                    
                    addTok = strtok(command, " ");
                    while(addTok != NULL)
                    {
                        if(strstr(addTok,"+=") > 0)
                        {
                            addV = 1;
                        }
                        else if(addV == 0)
                        {
                            current = head;
                            while(current != NULL)
                            {
                                if(strstr(addTok,current->name) > 0)
                                {
                                    strcpy(addVar,current->name);
                                    addval = atof(current->value);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        else if(addV == 1)
                        {
                            addval = addval + atof(addTok);
                            while(current != NULL)
                            {
                                if(strstr(addVar,current->name) > 0)
                                {
                                    sprintf(stringVal, "%2.2f", addval);
                                    strcpy(current->value,stringVal);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        addTok = strtok(NULL, " ");
                    }
                }
                else if(strstr(command,"-=") > 0)
                {
                    char *subTok;
                    int subV = 0;
                    float subval = 0;
                    char *subVar = (char *) malloc(1000);
                    char *stringVal = (char *) malloc(1000);
                    
                    subTok = strtok(command, " ");
                    while(subTok != NULL)
                    {
                        if(strstr(subTok,"-=") > 0)
                        {
                            subV = 1;
                        }
                        else if(subV == 0)
                        {
                            current = head;
                            while(current != NULL)
                            {
                                if(strstr(subTok,current->name) > 0)
                                {
                                    strcpy(subVar,current->name);
                                    subval = atof(current->value);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        else if(subV == 1)
                        {
                            subval = subval - atof(subTok);
                            while(current != NULL)
                            {
                                if(strstr(subVar,current->name) > 0)
                                {
                                    sprintf(stringVal, "%2.2f", subval);
                                    strcpy(current->value,stringVal);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        subTok = strtok(NULL, " ");
                    }
                }
                else if(strstr(command,"*=") > 0)
                {
                    char *mulTok;
                    int mulV = 0;
                    float mulval = 0;
                    char *mulVar = (char *) malloc(1000);
                    char *stringVal = (char *) malloc(1000);
                    
                    mulTok = strtok(command, " ");
                    while(mulTok != NULL)
                    {
                        if(strstr(mulTok,"*=") > 0)
                        {
                            mulV = 1;
                        }
                        else if(mulV == 0)
                        {
                            current = head;
                            while(current != NULL)
                            {
                                if(strstr(mulTok,current->name) > 0)
                                {
                                    strcpy(mulVar,current->name);
                                    mulval = atof(current->value);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        else if(mulV == 1)
                        {
                            mulval = mulval * atof(mulTok);
                            while(current != NULL)
                            {
                                if(strstr(mulVar,current->name) > 0)
                                {
                                    sprintf(stringVal, "%2.2f", mulval);
                                    strcpy(current->value,stringVal);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        mulTok = strtok(NULL, " ");
                    }
                    
                }
                else if(strstr(command,"/=") > 0)
                {
                    char *divTok;
                    int divV = 0;
                    float divval = 0;
                    char *divVar = (char *) malloc(1000);
                    char *stringVal = (char *) malloc(1000);
                    
                    divTok = strtok(command, " ");
                    while(divTok != NULL)
                    {
                        if(strstr(divTok,"/=") > 0)
                        {
                            divV = 1;
                        }
                        else if(divV == 0)
                        {
                            current = head;
                            while(current != NULL)
                            {
                                if(strstr(divTok,current->name) > 0)
                                {
                                    strcpy(divVar,current->name);
                                    divval = atof(current->value);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        else if(divV == 1)
                        {
                            divval = divval / atof(divTok);
                            while(current != NULL)
                            {
                                if(strstr(divVar,current->name) > 0)
                                {
                                    sprintf(stringVal, "%2.2f", divval);
                                    strcpy(current->value,stringVal);
                                    break;
                                }
                                else
                                    current = current->next;
                            }
                        }
                        divTok = strtok(NULL, " ");
                    }
                    
                }
                else if(strstr(command,"=") > 0)
                {
                    char *equals;
                    int equalsVal;
                    char *eqPLace;
                    int places = 0;
                    int size = 0;
                    int size_tok = 0;
                    char *toktok;
                    // Getting strings
                    if(strstr(command,"\"") > 0)
                    {
                        new = (link)malloc(sizeof(variables));
                        new->next = head;
                        head = new;
                        toktok = strtok(command," =");
                        
                        while(toktok != NULL)
                        {
                            if(size == 0)
                            {
                                size = strlen(toktok);
                                strcpy(new->name,toktok);
                            }
                            else if (size > 0 )
                            {strcpy(new->value,toktok);}
                            toktok = strtok(NULL, "=");
                        }
                        size = 0;
                        current = head;
                        while(current->next != NULL)
                        {current = current->next;}
                    }
                    else
                    {
                        // allocating for new variable
                        new = (link)malloc(sizeof(variables));
                        new->next = head;
                        head = new;
                        toktok = strtok(command,showsplit);
                        float add,min,mul,div;
                        float val;
                        //
                        while( toktok != NULL ) {
                            
                            if(strstr(toktok,"=") > 0)
                            {equalsVal = 1;}
                            else if(strstr(toktok,"+") > 0)
                            {add = 1;}
                            else if(strstr(toktok,"-") > 0)
                            {min = 1;}
                            else if(strstr(toktok,"*") > 0)
                            {mul = 1;}
                            else if(strstr(toktok,"/") > 0)
                            {div = 1;}
                            else
                            {
                                if (size == 0)
                                {
                                    size_tok = strlen(toktok);
                                    strcpy(new->name,toktok);
                                    size = strlen(new->name);
                                }
                                else if (size > 0)
                                {
                                    size_tok = strlen(toktok);
                                    char *string = (char *) malloc(1000);
                                    int count = 0;
                                    
                                    if (strstr(toktok,"?") > 0)
                                    {
                                        fgets(string,1000,stdin);
                                        strcpy(new->value,string);
                                        
                                    }
                                    else
                                    {
                                        
                                        if(add == 1)
                                        {
                                            
                                            if(isdigit(toktok[0]))
                                            {
                                                add = 0;
                                                val = atof(new->value) + atof(toktok);
                                                char *stringVal = (char *) malloc(1000);
                                                sprintf(stringVal, "%.2f", val);
                                                strcpy(new->value,stringVal);
                                            }
                                            else
                                            {
                                                add = 0;
                                                current = head;
                                                while(current != NULL)
                                                {
                                                    if(strstr(toktok,current->name) > 0)
                                                    {
                                                        val = atof(current->value) + atof(new->value);
                                                        char *stringVal = (char *) malloc(1000);
                                                        sprintf(stringVal, "%.2f", val);
                                                        strcpy(new->value,stringVal);
                                                        break;
                                                    }
                                                    else
                                                        current = current->next;
                                                }
                                            }
                                        }
                                        else if(min == 1)
                                        {
                                            if(isdigit(toktok[0]))
                                            {
                                                min = 0;
                                                val = atof(new->value) - atof(toktok);
                                                char *stringVal = (char *) malloc(1000);
                                                sprintf(stringVal, "%.2f", val);
                                                strcpy(new->value,stringVal);
                                            }
                                            else
                                            {
                                                min = 0;
                                                current = head;
                                                while(current != NULL)
                                                {
                                                    if(strstr(toktok,current->name) > 0)
                                                    {
                                                        val = atoi(new->value) - atof(current->value);
                                                        char *stringVal = (char *) malloc(1000);
                                                        sprintf(stringVal, "%.2f", val);
                                                        strcpy(new->value,stringVal);
                                                        break;
                                                    }
                                                    else
                                                        current = current->next;
                                                }
                                            }
                                        }
                                        else if(mul == 1)
                                        {
                                            if(isdigit(toktok[0]))
                                            {
                                                mul = 0;
                                                val = atof(new->value) * atof(toktok);
                                                char *stringVal = (char *) malloc(1000);
                                                sprintf(stringVal, "%.2f", val);
                                                strcpy(new->value,stringVal);
                                            }
                                            else
                                            {
                                                mul = 0;
                                                current = head;
                                                while(current != NULL)
                                                {
                                                    if(strstr(toktok,current->name) > 0)
                                                    {
                                                        val = atoi(new->value) * atof(current->value);
                                                        char *stringVal = (char *) malloc(1000);
                                                        sprintf(stringVal, "%.2f", val);
                                                        strcpy(new->value,stringVal);
                                                        break;
                                                    }
                                                    else
                                                        current = current->next;
                                                }
                                            }
                                        }
                                        else if(div == 1)
                                        {
                                            if(isdigit(toktok[0]))
                                            {
                                                mul = 0;
                                                val = atof(new->value) / atof(toktok);
                                                char *stringVal = (char *) malloc(1000);
                                                sprintf(stringVal, "%.2f", val);
                                                strcpy(new->value,stringVal);
                                            }
                                            else
                                            {
                                                div = 0;
                                                current = head;
                                                while(current != NULL)
                                                {
                                                    if(strstr(toktok,current->name) > 0)
                                                    {
                                                        val = atoi(new->value) / atof(current->value);
                                                        char *stringVal = (char *) malloc(1000);
                                                        sprintf(stringVal, "%.2f", val);
                                                        strcpy(new->value,stringVal);
                                                        break;
                                                    }
                                                    else
                                                        current = current->next;
                                                }
                                            }
                                        }
                                        else if(isdigit(toktok[0]))
                                        {
                                            while(count <= size_tok-1)
                                            {
                                                string[count] = toktok[count];
                                                count++;
                                            }
                                            strcpy(new->value,string);
                                        }
                                        else
                                        {
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(toktok,current->name) > 0)
                                                {
                                                    strcpy(new->value,current->value);
                                                    break;
                                                }
                                                else
                                                    current = current->next;
                                            }
                                            
                                        }
                                    }
                                    
                                }
                            }
                            toktok = strtok(NULL, showsplit);
                        }
                        current = head;
                        while(current->next != NULL)
                        {current = current->next;}
                        size = 0;
                        
                    }
                }
                else if(strstr(command,"[end]") > 0)
                {
                    puts("");
                }
                else
                    puts("");
                
            }
            free(0);
            return 0;
            
        }
        
        
        
        
        
        
        
        
        
        
        
        ///
        /// Interpreter
        /// Turtle Interpreter
        // Interpreter recieves a file
        // Documentation will either be under
        // This section or above under the interactive
        // interpreter mode
        
       
        
        
        
        
        
        
        
        
        // INTERPRETER
        if( (file = fopen(filename, "r+")) != NULL)
        {
           
           // Read file while not end of file
            while(!feof(file))
            {
               
                char showsplit[2] = " ";
                char *showtok;
                char *showtoks;
                // Display everything after except show
                float shaddnum = 0;
                float shaddtotal = 0;
                float shsubnum = 0;
                float shsubtotal = 0;
                float shmultinum = 1;
                float shmultitotal = 0;
                float shdivnum = 1.0;
                float shdivtotal = 0;
                float sum_token = 0;
                float number_token = 0;
                bool add_token = false;
                bool sub_token = false;
                bool multi_token = false;
                bool div_token = false;
                size_t len;
                int showValue = 0;
                int addvalue = 0;
                int subvalue = 0;
                int multivalue = 0;
                int divvalue = 0;
                int modvalue = 0;
                int equal;
                char *string;
                char *place;
                int place_Comment = 0;
                int place_show = 0;
                int count = 0;
                // token is a line
                token=fgets(line,1000,file);
                // Breaks if token is Null
                if(token == NULL)
                    break;
               else
               {
                   // Def function
                   // function is read
                   // Terminated the reading of function at [end]
                   // Call function by ()
                   if(strstr(token,"def") || functionTrue == true)
                   {
                       if(lineCounter == -1)
                       {
                           news = (chain)malloc(sizeof(func));
                           news->newFunc = heads;
                           heads = news;
                           functionTrue = true;
                           lineCounter = 0;
                           strcpy(news->funcName,token);
                           
                           
                       }
                       else if(lineCounter == 0)
                       {
                           strcpy(news->lines[lineCounter].line,token);
                           // printf("%s \n",news->lines[lineCounter].line);
                           lineCounter++;
                           
                       }
                       else if(lineCounter > 0)
                       {
                           if(strstr(token,"[end]") > 0 || strstr(token,"[end]\n") > 0)
                           {
                               strcpy(news->lines[lineCounter].line,token);
                              //  printf("%s \n",news->lines[lineCounter].line);
                               //lineCounter++;
                               currents = heads;
                               while(currents->newFunc != NULL)
                               {currents = currents->newFunc;}
                               lineCounter = -1;
                               functionTrue = false;
                           }
                           else
                           {
                               strcpy(news->lines[lineCounter].line,token);
                               lineCounter++;
                           }
                       }
                   }
                   else if(strstr(token,"()") > 0)
                   {
                       
                       int defcount = 0;
                       currents = heads;
                       while(currents != NULL)
                       {
                            //char *NameToken = (char *) malloc(1000);
                            //strcat(NameToken,"def ");
                           //strcat(NameToken,token);
                           
                           if(strstr(currents->funcName,token) > 0)
                           {
                               
                               while (!(strstr("[end]",currents->lines[defcount].line) > 0))
                               {
                                   //puts("here in function");
                                   char *stringToken = (char *) malloc(1000);
                                   strcpy(stringToken,currents->lines[defcount].line);
                                   //printf("%s",stringToken);
                                   interpreter(stringToken);
                                    defcount++;
                               }
                               break;
                           }
                           else
                           {
                              // puts("here");
                              // printf("%s",currents->funcName);
                               currents = currents->newFunc;
                           }
                       }
                   }
                   else if(strstr(token,"[if]") > 0 )
                   {
                       // [if] structure
                       // Reads structure
                       // Then evaluates the condition
                       // Then execute proper statements
                       
                        // puts("here 0");
                       ifStruct = true;
                       newss = (ifSLine)malloc(sizeof(ifStat));
                       newss->ifStates = headss;
                       headss = newss;
                       ifCounter = 1;
                       strcpy(newss->ifLine,token);
                       strcpy(newss->lines[0].line,token);
                       
                   }
                   else if(ifCounter == 1)
                   {
                        //puts("here 1");
                       strcpy(newss->lines[ifCounter].line,token);
                       ifCounter++;
                       
                   }
                   else if(ifCounter > 0)
                   {
                       // variable to check to make sure structure statment
                       // is true
   
                       
                       if(strstr(token,"[end]") > 0 || strstr(token,"[end]\n") > 0)
                       {
                          
                            //puts("here ending");
                           strcpy(newss->lines[ifCounter].line,token);
                           currentss = headss;
                           while(currentss->ifStates != NULL)
                           {currentss = currentss->ifStates;}
                           ifCounter = -1;
                           functionTrue = false;
                           
                           currentss = headss;
                           while(currentss != NULL)
                           {
                              
                               if(ifTrue == false)
                               {
                                   while(!(strstr("[end]",currentss->lines[ifLineCount].line) > 0))
                                   {
                        
                                       char *ConditionToken = (char *) malloc(1000);
                                       strcpy(ConditionToken,currentss->lines[ifLineCount].line);
                                
                                    if(strstr(ConditionToken,"[if]") > 0)
                                    {
                                        float floValue1;
                                        float floValue2;
                                        char  *strValue1;
                                        char  *strValue2;
                                        bool boValue1;
                                        bool boValue2;
                                        bool state;
                                        bool valueState =false;
                                        int which = -1;
                                        bool and,or,not =false;
                                        bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                        char *ifCondition = (char *) malloc(1000);
                                        ifCondition = strtok(ConditionToken," ");
                                        
                                        while(ifCondition != NULL)
                                        {
                                           //printf("%s ",ifCondition);
                                            if(strstr(ifCondition,"[if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"and") > 0)
                                                and = true;
                                            else if(strstr(ifCondition,"or") > 0)
                                                or = true;
                                            else if(strstr(ifCondition,"not") > 0)
                                                not = true;
                                            else if(strstr(ifCondition,"==") > 0)
                                                equal = true;
                                            else if(strstr(ifCondition,">") > 0)
                                                greater = true;
                                            else if(strstr(ifCondition,"<") > 0)
                                                less = true;
                                            else if(strstr(ifCondition,">=") > 0)
                                                greatEqual = true;
                                            else if(strstr(ifCondition,"<=") > 0)
                                                lessEqual = true;
                                            else if(strstr(ifCondition,"!=") > 0)
                                                notEqual = true;
                                            else
                                            {
                                                if(isdigit(ifCondition[0]) && valueState == false)
                                                {
                                                        which = 0;
                                                        floValue1 = atof(ifCondition);
                                                        valueState = true;
                                                }
                                                else if(isdigit(ifCondition[0]) && valueState == true)
                                                {
                                                        floValue2 = atof(ifCondition);
                                                        valueState = true;
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                                {       which = 1;
                                                        strcpy(strValue1,ifCondition);
                                                        valueState = true;
                                                
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                                {
                                                        strcpy(strValue2,ifCondition);
                                                        valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                                {
                                                        which = 2;
                                                        boValue1 = true;
                                                        valueState = true;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                                {
                                                        boValue1 = false;
                                                        valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                                {
                                                        boValue2 = true;
                                                        valueState = false;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                                {
                                                        boValue2 = false;
                                                        valueState = false;
                                                }
                                                else if(valueState == false)
                                                {
                                                    
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue1 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = false;
                                                                    break;
                                                                }
                                                                
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                    
                                                                    //printf("%c",current->value[count]);
                                                                        strValue1[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                //printf("%s",current->value);
                                                                //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                                else if(valueState == true)
                                                {
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue2 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = false;
                                                                    break;
                                                                }
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue2[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                            else
                                                                current = current->next;
                                                        }
                                                        valueState = true;
                                                }
                                            }
                                            
                                            
                                                ifCondition = strtok(NULL," ");
                                            }
                                        
                                            if(which == 0)
                                            {
                                                 if(equal == true)
                                                {
                                                    if(floValue1 == floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(greater == true)
                                                {
                                                     if(floValue1 == floValue2)
                                                        ifTrue = false;
                                                    else if(floValue1 > floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(less == true)
                                                {
                                                    if(floValue1 == floValue2)
                                                        ifTrue = false;
                                                    else if(floValue1 < floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(greatEqual == true)
                                                {
                                                    if(floValue1 >= floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(lessEqual == true)
                                                {
                                                    if(floValue1 <= floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(notEqual == true)
                                                {
                                                    if(floValue1 == floValue2)
                                                        ifTrue = false;
                                                    else if(floValue1 != floValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                
                                            }
                                            else if(which == 2)
                                            {
                                                if(and == true)
                                                {
                                                    if(boValue1 && boValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(or == true)
                                                {
                                                    if(boValue1 || boValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(not == true)
                                                {
                                                    if(!boValue1)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(equal == true)
                                                {
                                                    if(boValue1 == boValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(less == true)
                                                {
                                                    if(boValue1 < boValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(greater == true)
                                                {
                                                    if(boValue1 > boValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                            }
                                            else if(which == 1)
                                            {
                                                if(equal == true)
                                                {
                                                    if(strValue1 == strValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                }
                                                else if(greater == true)
                                                {
                                                    if(strValue1 > strValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                else if(less == true)
                                                {
                                                    if(strValue1 < strValue2)
                                                        ifTrue = true;
                                                    else
                                                        ifTrue = false;
                                                    
                                                }
                                                
                                            }
                                        
                                           if(ifTrue == true)
                                           {
                                               //puts("here");
                                               ifLineCount++;
                                               while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                               {
                                                   //printf("%s",currentss->lines[ifLineCount].line);
                                                   char *ConditionState = (char *) malloc(1000);
                                                   strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                                   interpreter(ConditionState);
                                                   //interpreter(currentss->lines[ifLineCount].line);
                                                   ifLineCount++;
                                                   //puts("here");
                                               }
                                               //ifTrue = false;
                                               //break;
                                           }
                                            else
                                            {
                                                ifLineCount++;
                                                while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                                   //printf("%s",currentss->lines[ifLineCount].line);
                                                    //interpreter(currentss->lines[ifLineCount].line);
                                                    ifLineCount++;
                                                    
                                                }
                                                ifTrue = false;
                                            }
                                    }
                                   else if(strstr(ConditionToken,"[then if]") > 0)
                                    {
                                        float floValue1;
                                        float floValue2;
                                        char  *strValue1;
                                        char  *strValue2;
                                        bool boValue1;
                                        bool boValue2;
                                        bool state;
                                        bool valueState =false;
                                        int which = -1;
                                        bool and,or,not =false;
                                        bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                        char *ifCondition = (char *) malloc(1000);
                                        
                                        ifCondition = strtok(ConditionToken," ");
                                        
                                        while(ifCondition != NULL)
                                        {
                                            //printf("%s ",ifCondition);
                                            if(strstr(ifCondition,"[if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"[then") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"if]") > 0)
                                                ifbool = true;
                                            else if(strstr(ifCondition,"and") > 0)
                                                and = true;
                                            else if(strstr(ifCondition,"or") > 0)
                                                or = true;
                                            else if(strstr(ifCondition,"not") > 0)
                                                not = true;
                                            else if(strstr(ifCondition,"==") > 0)
                                                equal = true;
                                            else if(strstr(ifCondition,">") > 0)
                                                greater = true;
                                            else if(strstr(ifCondition,"<") > 0)
                                                less = true;
                                            else if(strstr(ifCondition,">=") > 0)
                                                greatEqual = true;
                                            else if(strstr(ifCondition,"<=") > 0)
                                                lessEqual = true;
                                            else if(strstr(ifCondition,"!=") > 0)
                                                notEqual = true;
                                            else
                                            {
                                                if(isdigit(ifCondition[0]) && valueState == false)
                                                {
                                                    which = 0;
                                                    floValue1 = atof(ifCondition);
                                                    valueState = true;
                                                }
                                                else if(isdigit(ifCondition[0]) && valueState == true)
                                                {
                                                    floValue2 = atof(ifCondition);
                                                    valueState = false;
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                                {       which = 1;
                                                    strcpy(strValue1,ifCondition);
                                                    valueState = true;
                                                    
                                                }
                                                else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                                {
                                                    strcpy(strValue2,ifCondition);
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                                {
                                                    which = 2;
                                                    boValue1 = true;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                                {
                                                    boValue1 = false;
                                                    valueState = true;
                                                }
                                                else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                                {
                                                    boValue2 = true;
                                                    valueState = false;
                                                }
                                                else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                                {
                                                    boValue2 = false;
                                                    valueState = false;
                                                }
                                                else if(valueState == false)
                                                {
                                                    
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue1 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue1 = false;
                                                                    break;
                                                                }
                                                                
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue1[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                                else if(valueState == true)
                                                {
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(ifCondition,current->name) > 0)
                                                        {
                                                            if (isdigit(current->value[0]))
                                                            {
                                                                //printf("%s \n",current->value);
                                                                which = 0;
                                                                floValue2 = atof(current->value);
                                                                break;
                                                            }
                                                            else
                                                            {
                                                                if(strstr(current->value,"true") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = true;
                                                                    break;
                                                                }
                                                                else if(strstr(current->value,"false") >0)
                                                                {
                                                                    which = 2;
                                                                    boValue2 = false;
                                                                    break;
                                                                }
                                                                else
                                                                {
                                                                    which = 1;
                                                                    int size = 0;
                                                                    int count = 2;
                                                                    int counts = 0;
                                                                    size = strlen(current->value);
                                                                    while (count <= size-3)
                                                                    {
                                                                        
                                                                        //printf("%c",current->value[count]);
                                                                        strValue2[counts] = current->value[count];
                                                                        count++;
                                                                        counts++;
                                                                    }
                                                                    //printf("%s",current->value);
                                                                    //puts("");
                                                                    break;
                                                                }
                                                            }
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    valueState = true;
                                                }
                                            }
                                            
                                            
                                            ifCondition = strtok(NULL," ");
                                        }
                                        
                                        if(which == 0)
                                        {
                                            
                                            if(equal == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 > floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 < floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greatEqual == true)
                                            {
                                                if(floValue1 >= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(lessEqual == true)
                                            {
                                                if(floValue1 <= floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(notEqual == true)
                                            {
                                                if(floValue1 == floValue2)
                                                    ifTrue = false;
                                                else if(floValue1 != floValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                        }
                                        else if(which == 2)
                                        {
                                            if(and == true)
                                            {
                                                if(boValue1 && boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(or == true)
                                            {
                                                if(boValue1 || boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(not == true)
                                            {
                                                if(!boValue1)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(equal == true)
                                            {
                                                if(boValue1 == boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(less == true)
                                            {
                                                if(boValue1 < boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(greater == true)
                                            {
                                                if(boValue1 > boValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                        }
                                        else if(which == 1)
                                        {
                                            if(equal == true)
                                            {
                                                if(strValue1 == strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                            }
                                            else if(greater == true)
                                            {
                                                if(strValue1 > strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            else if(less == true)
                                            {
                                                if(strValue1 < strValue2)
                                                    ifTrue = true;
                                                else
                                                    ifTrue = false;
                                                
                                            }
                                            
                                        }
                                        
                                        if(ifTrue == true)
                                        {
                                            //puts("here");
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                            {
                                                //printf("%s",currentss->lines[ifLineCount].line);
                                                char *ConditionStates = (char *) malloc(1000);
                                                strcpy(ConditionStates,currentss->lines[ifLineCount].line);
                                                interpreter(ConditionStates);
                                                ifLineCount++;
                                                //puts("here");
                                            }
                                            //ifTrue = false;
                                            //break;
                                            
                                        }
                                        else
                                        {
                                            ifLineCount++;
                                            while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                                // printf("%s",currentss->lines[ifLineCount].line);
                                                //interpreter(currentss->lines[ifLineCount].line);
                                                ifLineCount++;
                                                
                                            }
                                            ifTrue = false;
                                        }
                                    }
                                    
                                    else if(strstr(ConditionToken,"[then]") > 0)
                                    {
                                   
                                        ifTrue = true;
                                                
                                                
                                                if(ifTrue == true)
                                                {
                                                    ifLineCount++;
                                                    while (!(strstr(currentss->lines[ifLineCount].line,"[end]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[ended]") > 0)) {
                                                        char *ConditionState = (char *) malloc(1000);
                                                        strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                                        interpreter(ConditionState);
                                                        ifLineCount++;
                                                        
                                                    }
                                                    //ifTrue = false;
                                                    break;
                                                    
                                                }
                                                else
                                                    ifTrue = true;
                                                
                                    }
                                       if(ifTrue == true)
                                           break;
                                       else
                                           continue;
                                       
                                       
                                            
                                    }
                               }
                               else
                                   break;
                               
                           }
                       }
                       else
                       {
                           
                           //puts("here added");
                           strcpy(newss->lines[ifCounter].line,token);
                           ifCounter++;
                       }
                   }
                   else if(strstr(token,"[loop]") >0)
                   {
                       // LOOP structure
                       // read the structure
                       // then execute the structure
                       loopBool = true;
                       newed = (looper)malloc(sizeof(loops));
                       newed->newloop = headed;
                       headed = newed;
                       loopCount = 1;
                       strcpy(newed->loopName,token);
                       char *loopToken;
                       
                       //puts("here loop");
                       loopToken = strtok(token," ");
                       while(loopToken != NULL)
                       {
                           if(isdigit(loopToken[0]))
                               loopIncrement = atoi(loopToken);
                           
                           loopToken = strtok(NULL," ");
                       }
                       
                       
                   }
                   else if(loopCount == 1)
                   {
                       //puts("here added first loop");
                       strcpy(newed->lines[loopCount].line,token);
                       loopCount++;
                   }
                   else if(loopCount > 0)
                   {
                       // puts("here added more than one");
                       if(strstr(token,"[end]") > 0 || strstr(token,"[end]\n") > 0)
                       {
                           strcpy(newed->lines[loopCount].line,token);
                           
                           currented = headed;
                           while(currented->newloop != NULL)
                           {currented = currented->newloop;}
                           
                           int counters = 0;
                           int count = 0;
                           while (count < loopIncrement)
                           {
                               counters = 0;
                               while(counters < loopCount)
                               {
                                  // puts("here added more than one");
                                   char *stringToken = (char *) malloc(1000);
                                   strcpy(stringToken,currented->lines[counters].line);
                                   interpreter(stringToken);
                                   counters++;
                               }
                               count++;
                           }
                           loopCount = -1;
                    
                       }
                       else
                       {
                            strcpy(newed->lines[loopCount].line,token);
                           loopCount++;
                           
                       }
                   }
                   // show function
                   else if (strstr(token,"show") > 0)
                   {
                       // Display expressions, variables, strings
                       string = strstr(token,"//");
                       place = strstr(token,"show");
                       place_Comment = string - token;
                       place_show = place - token;
                       // Dealing with comments
                       if(place_Comment < place_show)
                           continue;
                       else{
                       
                       // Will do math if it finds an expression
                       if(strstr(token,"\"") > 0)
                       {
                           
                           //showtok = strtok(token,"\"");
                           showtok = strtok(token,"\"");
                           while (showtok != NULL) {
                               
                               //printf("%s \n",showtok);
                               if (strstr(showtok,"show") > 0)
                                   showValue = 1;
                               else
                               {
                                   // Display Strings
                                   if (strstr(showtok,"\""))
                                   {
                                       char *stringArray = (char *) malloc(1000);
                                       int count =0;
                                       while(showtok[count] != '\"')
                                       {
                                           stringArray[count] = showtok[count];
                                           count++;
                                       }
                                       printf("%s ",stringArray);
                                       free(stringArray);
                                   }
                                   else
                                       printf("%s ",showtok);
                               }
                               showtok = strtok(NULL,showsplit);
                           }
                          printf("\n");
                       }
                      else if(strstr(token,"+") > 0  || strstr(token,"-") > 0 || strstr(token,"*") > 0 || strstr(token,"/") > 0)
                       {
                           // Display numbers and math expressions
                           showtok = strtok(token," ");
                           while (showtok != NULL) {
                           
                               if (strstr(showtok,"show") > 0)
                                   showValue = 1;
                               else if (strstr(showtok,"+") > 0)
                                   add_token = true;
                               else if (strstr(showtok,"-") > 0)
                                   sub_token = true;
                               else if (strstr(showtok,"*") > 0)
                                   multi_token = true;
                               else if (strstr(showtok,"/") > 0)
                                   div_token = true;
                               else if(isnumber(showtok[0]))
                               {
                                   if (number_token == 0)
                                   {
                                       if (multi_token == true)
                                       {
                                           number_token = atof(showtok);
                                           sum_token = 1;
                                       }
                                       else if(div_token == true)
                                       {
                                           number_token = atof(showtok);
                                           sum_token = 1;
                                       }
                                       else
                                           number_token = atof(showtok);
                                   }
                                   else
                                       sum_token = atof(showtok);
                           
                                   if(add_token == true)
                                   {
                                       number_token += sum_token;
                                       add_token = false;
                                   }
                                   else if(sub_token == true)
                                   { number_token -= sum_token;
                                       sub_token = false;
                                   }
                                   else if(multi_token == true)
                                   {
                                       number_token *= sum_token;
                                       multi_token = false;
                                   }
                                   else if(div_token == true)
                                   {   number_token /= sum_token;
                                       div_token = false;
                                   }
                               }
                                showtok = strtok(NULL,showsplit);
                           }
                           // check for decimal place
                           // add that
                            printf("%3.3f \n",number_token);
                            number_token = 0;
                            sum_token = 0;
                           
                       }
                      else if(strstr(token,"%") > 0)
                      {
                          // Display modulus
                          int shnum0 =0;
                          int shnums1 =0;
                          showtok = strtok(token,showsplit);
                          while (showtok != NULL) {
                            // printf("%s \n",showtok );
                              if (strstr(showtok,"%") > 0)
                                  //printf("%s \n",showtok );
                                  modvalue = 1;
                              else if(strstr(showtok,"show") > 0)
                                  modvalue = 1;
                              else if (shnum0 == 0)
                                  shnum0 = atoi(showtok);
                              else if (shnum0 > 0 && shnums1 == 0)
                                  shnums1 = atoi(showtok);
                              
                              showtok = strtok(NULL," ");
                          }
                          shnum0 %= shnums1;
                          printf("%d \n",shnum0);
                      }
                      else
                      {
                          // Display variables
                          showtok = strtok(token,showsplit);
                         while (showtok != NULL) {
                             
                              if (strstr(showtok,"show") > 0)
                             {
                                 showValue = 1;
                             }
                              else
                              {
                                  current = head;
                                  while(current != NULL)
                                  {
                                      if(strstr(showtok,current->name) > 0)
                                      {
                                          if (isdigit(current->value[0]))
                                          {
                                              printf("%s \n",current->value);
                                              break;
                                          }
                                          else
                                          {
                                              int size = 0;
                                              int count = 2;
                                              size = strlen(current->value);
                                              while (current->value[count] != '\"') {
                                                  printf("%c",current->value[count]);
                                                  count++;
                                              }
                                              //printf("%s",current->value);
                                              puts("");
                                              break;
                                          }
                                      }
                                      else
                                          current = current->next;
                                  }
                              }
                               showtok = strtok(NULL,showsplit);
                          }
                      }
                    }
                   }
                   else if(strstr(token,"//") > 0)
                   {
                       // Comments
                       // continue if // comments
                       continue;
                   }
                   else if (strstr(token,"sys") > 0)
                   {
                       // sys commands
                       //token = strncpy(token,"   ",3);
                      // system(&token[4]);
                       
                       int sys_value = 0;
                       char *sys_token = (char *) malloc(1000);
                       showtok = strtok(token," ");
                       while (showtok != NULL) {
                       
                           if (strstr(showtok, "sys") > 0)
                               sys_value = 1;
                           else
                           {
                               strcat(sys_token,showtok);
                               strcat(sys_token," ");
                               //printf("%s ",sys_token);
                           }
                           showtok = strtok(NULL,showsplit);
                       }
                       system(sys_token);
                   }
                   else if(strstr(token,"[exit]") > 0)
                   {
                       // Exit function
                       exit(0);
                   }
                   else if(strstr(token,"[r]") > 0)
                   {
                      // Read file
                       // file will display after being read
                       FILE *readFile;
                       char *readingFile  = (char *) malloc(1000);
                       int read_value = 0;
                       char *read_token;
                       char *readLine  = (char *) malloc(1000);
                       
                       read_token = strtok(token," ");
                       while(read_token != NULL)
                       {
                           if (strstr(read_token,"[r]") > 0)
                           {
                               read_value = 1;
                           }
                           else
                           {
                               int length = 0;
                               int count = 0;
                               length = strlen(read_token);
                              
                               while(count < length)
                               {
                                   readingFile[count] =  read_token[count];
                                   count++;
                               }
                
                               char lineRead[1000];
                               if( (readFile = fopen(readingFile, "r+")) != NULL)
                               {
                                   while(!feof(readFile))
                                   {
                                        readLine =fgets(lineRead,1000,readFile);
                                       if (readLine == NULL)
                                           continue;
                                       else
                                            printf("%s",readLine);
                                   }
                               }
                                else if( (readFile = fopen(readingFile, "r+")) == NULL)
                                {
                                    printf("%s not found \n",readingFile);
                                }
                               
                           }
                            read_token = strtok(NULL," ");
                       }
                       puts("");
                       
                   }
                   else if(strstr(token,"[w]") > 0)
                   {
                       // write file
                       FILE  *writeFile;
                       char *writingFile = (char *) malloc(1000);
                       int write_Value = 0;
                       char *write_token;
                       char *writeLine = (char *) malloc(1000);;
                       char *delimator = "\n";
                       write_token = strtok(token," ");
                       
                       while(write_token != NULL)
                       {
                           if(strstr(write_token,"[w]") > 0)
                               write_Value = 1;
                           else
                           {
                               //writingFile = write_token;
                               // printf("%s",write_token);
                               
                               
                               int length = 0;
                               int count = 0;
                               length = strlen(write_token);
                               
                               while(count < length)
                               {
                                   writingFile[count] =  write_token[count];
                                   count++;
                               }
                               if( (writeFile  = fopen(writingFile, "w+")) != NULL )
                               {
                                   while (!strstr(writeLine,delimator)) {
                                      
                                       fprintf(writeFile,"%s" ,fgets(writeLine,1000,stdin));
                                   }
                                  
                                   fclose(writeFile);
                               }
                               else if ((writeFile  = fopen(writingFile, "w+")) == NULL)
                               {
                                   printf("%s not found \n",writingFile);
                               }
                               
                               
                           }
                           
                            write_token = strtok(NULL," ");
                       }
                   }
                   else if (strstr(token,"[a]") > 0)
                   {
                       // appending a file
                       FILE  *appendFile;
                       char *appendingFile = (char *) malloc(1000);
                       int append_Value = 0;
                       char *append_token;
                       char *appendLine = (char *) malloc(1000);;
                       char *delimator = "\n";
                       append_token = strtok(token," ");
                       
                       while(append_token != NULL)
                       {
                           if(strstr(append_token,"[a]") > 0)
                               append_Value = 1;
                           else
                           {
                               //appendingFile = append_token;
                                //printf("%s",append_token);
                               int length = 0;
                               int count = 0;
                               length = strlen(append_token);
                               
                               while(count < length)
                               {
                                   appendingFile[count] =  append_token[count];
                                   count++;
                               }
                               if( (appendFile  = fopen(appendingFile, "a+")) != NULL )
                               {
                                   while (!strstr(appendLine,delimator)) {
                                       
                                       fprintf(appendFile,"%s" ,fgets(appendLine,1000,stdin));
                                   }
                                   
                                   fclose(appendFile);
                               }
                               else if ((appendFile  = fopen(appendingFile, "a+")) == NULL)
                               {
                                   printf("%s not found \n",appendingFile);
                               }
                           }
                           
                           append_token = strtok(NULL," ");
                       }
                   }
                   else if(strstr(token,"[math]") > 0)
                   {
                       // math operations
                       char showsplit[2] = " ";
                       char *showtok;
                       char *showtoks;
                       // Display everything after except show
                       float shaddnum = 0;
                       float shaddtotal = 0;
                       float shsubnum = 0;
                       float shsubtotal = 0;
                       float shmultinum = 1;
                       float shmultitotal = 0;
                       float shdivnum = 1.0;
                       float shdivtotal = 0;
                       float sum_token = 0;
                       float number_token = 0;
                       bool add_token = false;
                       bool sub_token = false;
                       bool multi_token = false;
                       bool div_token = false;
                       int shnum0 = 0;
                       int shnums1 = 0;
                       int showValue = 0;
                       int addvalue = 0;
                       int subvalue = 0;
                       int multivalue = 0;
                       int divvalue = 0;
                       int modvalue = 0;
                       int equal;
                       int count = 0;
                       
                       
                        if(strstr(token,"+") > 0  || strstr(token,"-") > 0 || strstr(token,"*") > 0 || strstr(token,"/") > 0)
                       {
                           showtok = strtok(token," ");
                           while (showtok != NULL) {
                               
                               if (strstr(showtok,"[math]") > 0)
                                   showValue = 1;
                               else if (strstr(showtok,"+") > 0)
                                   add_token = true;
                               else if (strstr(showtok,"-") > 0)
                                   sub_token = true;
                               else if (strstr(showtok,"*") > 0)
                                   multi_token = true;
                               else if (strstr(showtok,"/") > 0)
                                   div_token = true;
                               else if(isnumber(showtok[0]))
                               {
                                   if (number_token == 0)
                                   {
                                       if (multi_token == true)
                                       {
                                           number_token = atof(showtok);
                                           sum_token = 1;
                                       }
                                       else if(div_token == true)
                                       {
                                           number_token = atof(showtok);
                                           sum_token = 1;
                                       }
                                       else
                                           number_token = atof(showtok);
                                   }
                                   else
                                       sum_token = atof(showtok);
                                   
                                   if(add_token == true)
                                   {
                                       number_token += sum_token;
                                       add_token = false;
                                   }
                                   else if(sub_token == true)
                                   { number_token -= sum_token;
                                       sub_token = false;
                                   }
                                   else if(multi_token == true)
                                   {
                                       number_token *= sum_token;
                                       multi_token = false;
                                   }
                                   else if(div_token == true)
                                   {   number_token /= sum_token;
                                       div_token = false;
                                   }
                               }
                               showtok = strtok(NULL,showsplit);
                           }
                           // check for decimal place
                           // add that
                           printf("%3.3f \n",number_token);
                           number_token = 0;
                           sum_token = 0;
                       }
                        else if(strstr(token,"%") > 0)
                        {
                            
                            showtok = strtok(token,showsplit);
                            while (showtok != NULL) {
                                
                                if (strstr(showtok,"%") > 0)
                                    //printf("%s \n",showtok );
                                    modvalue = 1;
                                else if (shnum0 == 0)
                                    shnum0 = atoi(showtok);
                                else if (shnum0 != 0)
                                {shnums1 = atoi(showtok);}
                                showtok = strtok(NULL,showsplit);
                            }
                            shnum0 %= shnums1;
                            printf("%d \n",shnum0);
                            shnum0 = 0;
                            shnums1 = 0;
                        }
                       else
                       {
                           puts("no operator or invalid operator");
                           break;
                       }
                       
                   }
                   else if (strstr(token,"clear") > 0)
                   {
                       // clear screen
                       system(token);
                   }
                   else if (strstr(token,"[mkdir]") > 0)
                   {
                       
                       // Making Directory
                       char *mkdir = (char *) malloc(1000);
                       char *mkdir_token;
                       int mkdir_value = 0;
                       
                       mkdir_token = strtok(token," ");
                       while (mkdir_token != NULL)
                       {
                           if(strstr(mkdir_token,"[mkdir]") > 0)
                               mkdir_value = 1;
                           else
                           {
                               strcat(mkdir,"mkdir ");
                               strcat(mkdir,mkdir_token);
                           }
                            mkdir_token = strtok(NULL," ");
                           
                       }
                       system(mkdir);
                       
                   }
                   else if(strstr(token,"[remove]") > 0)
                   {
                       // Remove
                       char *rmdir = (char *) malloc(1000);
                       char *rmdir_token;
                       int rmdir_value = 0;
                       
                       rmdir_token = strtok(token," ");
                       while (rmdir_token != NULL)
                       {
                           if(strstr(rmdir_token,"[remove]") > 0)
                               rmdir_value = 1;
                           else
                           {
                               strcat(rmdir,"rmdir ");
                               strcat(rmdir,rmdir_token);
                           }
                           rmdir_token = strtok(NULL," ");
                       }
                    
                       system(rmdir);
                   }
                   else if(strstr(token,"[copy]") > 0)
                   {
                       // Copy a file or folder
                       
                      
                       char *cp = (char *) malloc(1000);
                       char *copy_token = (char *) malloc(1000);
                       char *destName = (char *) malloc(1000);
                       char *newDestname = (char *) malloc(1000);
                       char *dash = (char *) malloc(1000);
                       int copy_value = 0;
                       int returnOld;
                       int returnNew;
                       FILE *old,*new;
                       int ch;
                       int count = 0;
                       char *new_token;
                       
                       new_token = token;
                       while (new_token[count] != ' ')
                       {
                           copy_token[count] = new_token[count];
                           count++;
                       }
                      
                       count++;
                       int newCount = 0;
                      
                       while (new_token[count] != ' ')
                       {
                           destName[newCount] = new_token[count];
                           count++;
                           newCount++;
                       }
                     
                       newCount = 0;
                       
                       while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
                       {
                           dash[newCount] = new_token[count];
                           count++;
                           newCount++;
                       }
                       newCount = 0;
                       
                       while (new_token[count] != '\n')
                       {
                               newDestname[newCount] = new_token[count];
                           
                           count++;
                           newCount++;
                       }
                       
                       if ((old = fopen(destName,"rb")) == NULL )
                           returnOld = -1;
                       
                       if ((new = fopen(newDestname,"wb")) == NULL )
                       {
                           fclose(old);
                           returnNew = -1;
                       }
                       
                       if (returnOld == -1)
                       {
                           printf("%s not found \n",destName);
                           return 0;
                       }
                       else if (returnNew == -1)
                       {
                           printf("%s not found \n",newDestname);
                           return 0;
                       }
                       else
                       {
                           while(1)
                           {
                               ch = fgetc(old);
                               
                               if(!feof(old))
                                   fputc(ch,new);
                               else
                                   break;
                           }
                           
                           fclose(new);
                           fclose(old);
                       }
                       
                   }
                   else if (strstr(token,"[move]") > 0)
                   {
                       // move the file from one location to another
                       // Probably needs to be fixed
                       char *copy_token = (char *) malloc(1000);
                       char *destName = (char *) malloc(1000);
                       char *newDestname = (char *) malloc(1000);
                       char *dash = (char *) malloc(1000);
                       int copy_value = 0;
                       int returnOld;
                       int returnNew;
                       FILE *old,*new;
                       int ch;
                       int count = 0;
                       char *new_token;
                      
                       
                       new_token = token;
                       while (new_token[count] != ' ')
                       {
                           copy_token[count] = new_token[count];
                           count++;
                       }
                      
                       count++;
                       int newCount = 0;
                       
                       while (new_token[count] != ' ')
                       {
                           destName[newCount] = new_token[count];
                           count++;
                           newCount++;
                       }
                       newCount = 0;
                       
                       while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
                       {
                           dash[newCount] = new_token[count];
                           count++;
                           newCount++;
                       }
                       newCount = 0;
                       
                       while (new_token[count] != '\n')
                       {
                           newDestname[newCount] = new_token[count];
                           count++;
                           newCount++;
                       }
                      
                       if ((old = fopen(destName,"rb")) == NULL )
                           returnOld = -1;
                       
                       if ((new = fopen(newDestname,"wb")) == NULL )
                       {
                           fclose(old);
                           returnNew = -1;
                       }
                       
                       if (returnOld == -1)
                       {
                           printf("%s not found \n",destName);
                           return 0;
                       }
                       else if (returnNew == -1)
                       {
                           printf("%s not found \n",newDestname);
                           return 0;
                       }
                       else
                       {
                           while(1)
                           {
                               ch = fgetc(old);
                               
                               if(!feof(old))
                                   fputc(ch,new);
                               else
                                   break;
                               
                           }
                           
                           fclose(new);
                           fclose(old);
                       }
                       
                    
                       remove(destName);
                       
                   }
                   else if(strstr(token,"+=") > 0)
                   {
                       // Incrementing
                       // a variable
                       // resolves the issue
                       // A = A + 1
                       // instead of that
                       // A += 1
                       // Same structure as decrementing and etc
                       char *addTok;
                       int addV = 0;
                       float addval = 0;
                       char *addVar = (char *) malloc(1000);
                       char *stringVal = (char *) malloc(1000);
                       
                       addTok = strtok(token, " ");
                      while(addTok != NULL)
                      {
                          if(strstr(addTok,"+=") > 0)
                          {
                              addV = 1;
                          }
                          else if(addV == 0)
                          {
                              current = head;
                              while(current != NULL)
                              {
                                  if(strstr(addTok,current->name) > 0)
                                  {
                                      strcpy(addVar,current->name);
                                      addval = atof(current->value);
                                      break;
                                  }
                                  else
                                      current = current->next;
                              }
                          }
                          else if(addV == 1)
                          {
                              addval = addval + atof(addTok);
                              while(current != NULL)
                              {
                                  if(strstr(addVar,current->name) > 0)
                                  {
                                      sprintf(stringVal, "%2.2f", addval);
                                      strcpy(current->value,stringVal);
                                      break;
                                  }
                                  else
                                      current = current->next;
                              }
                          }
                           addTok = strtok(NULL, " ");
                      }
                   }
                   else if(strstr(token,"-=") > 0)
                   {
                       char *subTok;
                       int subV = 0;
                       float subval = 0;
                       char *subVar = (char *) malloc(1000);
                       char *stringVal = (char *) malloc(1000);
                       
                       subTok = strtok(token, " ");
                       while(subTok != NULL)
                       {
                           if(strstr(subTok,"-=") > 0)
                           {
                               subV = 1;
                           }
                           else if(subV == 0)
                           {
                               current = head;
                               while(current != NULL)
                               {
                                   if(strstr(subTok,current->name) > 0)
                                   {
                                       strcpy(subVar,current->name);
                                       subval = atof(current->value);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           else if(subV == 1)
                           {
                               subval = subval - atof(subTok);
                               while(current != NULL)
                               {
                                   if(strstr(subVar,current->name) > 0)
                                   {
                                       sprintf(stringVal, "%2.2f", subval);
                                       strcpy(current->value,stringVal);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           subTok = strtok(NULL, " ");
                       }
                   }
                   else if(strstr(token,"*=") > 0)
                   {
                       char *mulTok;
                       int mulV = 0;
                       float mulval = 0;
                       char *mulVar = (char *) malloc(1000);
                       char *stringVal = (char *) malloc(1000);
                       
                       mulTok = strtok(token, " ");
                       while(mulTok != NULL)
                       {
                           if(strstr(mulTok,"*=") > 0)
                           {
                               mulV = 1;
                           }
                           else if(mulV == 0)
                           {
                               current = head;
                               while(current != NULL)
                               {
                                   if(strstr(mulTok,current->name) > 0)
                                   {
                                       strcpy(mulVar,current->name);
                                       mulval = atof(current->value);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           else if(mulV == 1)
                           {
                               mulval = mulval * atof(mulTok);
                               while(current != NULL)
                               {
                                   if(strstr(mulVar,current->name) > 0)
                                   {
                                       sprintf(stringVal, "%2.2f", mulval);
                                       strcpy(current->value,stringVal);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           mulTok = strtok(NULL, " ");
                       }
                       
                   }
                   else if(strstr(token,"/=") > 0)
                   {
                       char *divTok;
                       int divV = 0;
                       float divval = 0;
                       char *divVar = (char *) malloc(1000);
                       char *stringVal = (char *) malloc(1000);
                       
                       divTok = strtok(token, " ");
                       while(divTok != NULL)
                       {
                           if(strstr(divTok,"/=") > 0)
                           {
                               divV = 1;
                           }
                           else if(divV == 0)
                           {
                               current = head;
                               while(current != NULL)
                               {
                                   if(strstr(divTok,current->name) > 0)
                                   {
                                       strcpy(divVar,current->name);
                                       divval = atof(current->value);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           else if(divV == 1)
                           {
                               divval = divval / atof(divTok);
                               while(current != NULL)
                               {
                                   if(strstr(divVar,current->name) > 0)
                                   {
                                       sprintf(stringVal, "%2.2f", divval);
                                       strcpy(current->value,stringVal);
                                       break;
                                   }
                                   else
                                       current = current->next;
                               }
                           }
                           divTok = strtok(NULL, " ");
                       }
                       
                   }
                   else if(strstr(token,"=") > 0 )
                   {
                       // setting a value
                       // string, digit and etc
                       char *equals;
                       int equalsVal;
                       char *eqPLace;
                       int places = 0;
                       int size = 0;
                       int size_tok = 0;
                       char *toktok;
                       // Getting strings
                       if(strstr(token,"\"") > 0)
                       {
                           new = (link)malloc(sizeof(variables));
                           new->next = head;
                           head = new;
                           toktok = strtok(token," =");
                           
                           while(toktok != NULL)
                           {
                               if(size == 0)
                               {
                                   size = strlen(toktok);
                                   strcpy(new->name,toktok);
                               }
                               else if (size > 0 )
                               {strcpy(new->value,toktok);}
                               toktok = strtok(NULL, "=");
                           }
                           size = 0;
                           current = head;
                           while(current->next != NULL)
                           {current = current->next;}
                       }
                       else
                       {
                           // allocating for new variable
                           new = (link)malloc(sizeof(variables));
                           new->next = head;
                           head = new;
                           toktok = strtok(token,showsplit);
                           float add,min,mul,div;
                           float val;
                           //
                           while( toktok != NULL ) {
                               
                               if(strstr(toktok,"=") > 0)
                               {equalsVal = 1;}
                               else if(strstr(toktok,"+") > 0)
                               {add = 1;}
                               else if(strstr(toktok,"-") > 0)
                               {min = 1;}
                               else if(strstr(toktok,"*") > 0)
                               {mul = 1;}
                               else if(strstr(toktok,"/") > 0)
                               {div = 1;}
                               else
                               {
                                   if (size == 0)
                                   {
                                       size_tok = strlen(toktok);
                                       strcpy(new->name,toktok);
                                       size = strlen(new->name);
                                   }
                                   else if (size > 0)
                                   {
                                       size_tok = strlen(toktok);
                                       char *string = (char *) malloc(1000);
                                       int count = 0;
                                       
                                       if (strstr(toktok,"?") > 0)
                                       {
                                             fgets(string,1000,stdin);
                                            strcpy(new->value,string);
                                    
                                       }
                                       else
                                       {
                                           
                                               if(add == 1)
                                               {
                                                   
                                                   if(isdigit(toktok[0]))
                                                   {
                                                        add = 0;
                                                       val = atof(new->value) + atof(toktok);
                                                       char *stringVal = (char *) malloc(1000);
                                                       sprintf(stringVal, "%.2f", val);
                                                       strcpy(new->value,stringVal);
                                                   }
                                                   else
                                                   {
                                                       add = 0;
                                                       current = head;
                                                       while(current != NULL)
                                                       {
                                                           if(strstr(toktok,current->name) > 0)
                                                           {
                                                               if(isdigit(current->value[0]))
                                                               {
                                                                   val = atof(current->value) + atof(new->value);
                                                                   char *stringVal = (char *) malloc(1000);
                                                                   sprintf(stringVal, "%.2f", val);
                                                                   strcpy(new->value,stringVal);
                                                                   break;
                                                                   
                                                               }
                                                               else
                                                               {
                                                                   // ADDING STRINGS
                                                                   break;
                                                               }
                                                           }
                                                           else
                                                               current = current->next;
                                                       }
                                                   }
                                               }
                                               else if(min == 1)
                                               {
                                                   if(isdigit(toktok[0]))
                                                   {
                                                       min = 0;
                                                       val = atof(new->value) - atof(toktok);
                                                       char *stringVal = (char *) malloc(1000);
                                                       sprintf(stringVal, "%.2f", val);
                                                       strcpy(new->value,stringVal);
                                                   }
                                                   else
                                                   {
                                                       min = 0;
                                                       current = head;
                                                       while(current != NULL)
                                                       {
                                                           if(strstr(toktok,current->name) > 0)
                                                           {
                                                               val = atoi(new->value) - atof(current->value);
                                                               char *stringVal = (char *) malloc(1000);
                                                               sprintf(stringVal, "%.2f", val);
                                                               strcpy(new->value,stringVal);
                                                               break;
                                                           }
                                                           else
                                                               current = current->next;
                                                       }
                                                   }
                                               }
                                               else if(mul == 1)
                                               {
                                                   if(isdigit(toktok[0]))
                                                   {
                                                       mul = 0;
                                                       val = atof(new->value) * atof(toktok);
                                                       char *stringVal = (char *) malloc(1000);
                                                       sprintf(stringVal, "%.2f", val);
                                                       strcpy(new->value,stringVal);
                                                   }
                                                   else
                                                   {
                                                       mul = 0;
                                                       current = head;
                                                       while(current != NULL)
                                                       {
                                                           if(strstr(toktok,current->name) > 0)
                                                           {
                                                               val = atoi(new->value) * atof(current->value);
                                                               char *stringVal = (char *) malloc(1000);
                                                               sprintf(stringVal, "%.2f", val);
                                                               strcpy(new->value,stringVal);
                                                               break;
                                                           }
                                                           else
                                                               current = current->next;
                                                       }
                                                   }
                                               }
                                               else if(div == 1)
                                               {
                                                   if(isdigit(toktok[0]))
                                                   {
                                                       mul = 0;
                                                       val = atof(new->value) / atof(toktok);
                                                       char *stringVal = (char *) malloc(1000);
                                                       sprintf(stringVal, "%.2f", val);
                                                       strcpy(new->value,stringVal);
                                                   }
                                                   else
                                                   {
                                                       div = 0;
                                                       current = head;
                                                       while(current != NULL)
                                                       {
                                                           if(strstr(toktok,current->name) > 0)
                                                           {
                                                               val = atoi(new->value) / atof(current->value);
                                                               char *stringVal = (char *) malloc(1000);
                                                               sprintf(stringVal, "%.2f", val);
                                                               strcpy(new->value,stringVal);
                                                               break;
                                                           }
                                                           else
                                                               current = current->next;
                                                       }
                                                   }
                                               }
                                               else if(isdigit(toktok[0]))
                                               {
                                                   while(count <= size_tok-1)
                                                   {
                                                       string[count] = toktok[count];
                                                       count++;
                                                   }
                                                   strcpy(new->value,string);
                                               }
                                                else
                                                {
                                                    current = head;
                                                    while(current != NULL)
                                                    {
                                                        if(strstr(toktok,current->name) > 0)
                                                        {
                                                            strcpy(new->value,current->value);
                                                            break;
                                                        }
                                                        else
                                                            current = current->next;
                                                    }
                                                    
                                                }
                                       }
                                       
                                   }
                               }
                               toktok = strtok(NULL, showsplit);
                           }
                           current = head;
                           while(current->next != NULL)
                           {current = current->next;}
                           size = 0;
                       }
                   }
                   else
                   {
                       // Add to statement for
                       if(strlen(token) > 1)
                       {
                           printf("Invalid Statement %s ",token);
                       }
                       else
                          // printf("%s",token);
                           continue;
                   }
               }
            }
            // close file once interpreted
            fclose(file);
            break;
        }
        else if ((file = fopen(filename, "r+")) == NULL)
        {
            puts("File not found");
            break;
        }
        
    
    }
    
    free(0);
    return 0;
}



// Function
void interpreter(char *tokens)
{
    
    // character limit
    char line[1000];
    // token to be executed
    char showsplit[2] = " ";
    char *showtok = (char *) malloc(1000);
    char *showtoks = (char *) malloc(1000);
    // Display everything after except show
    float shaddnum = 0;
    float shaddtotal = 0;
    float shsubnum = 0;
    float shsubtotal = 0;
    float shmultinum = 1;
    float shmultitotal = 0;
    float shdivnum = 1.0;
    float shdivtotal = 0;
    float sum_token = 0;
    float number_token = 0;
    bool add_token = false;
    bool sub_token = false;
    bool multi_token = false;
    bool div_token = false;
    size_t len;
    int showValue = 0;
    int addvalue = 0;
    int subvalue = 0;
    int multivalue = 0;
    int divvalue = 0;
    int modvalue = 0;
    int equal;
    char *string = (char *) malloc(1000);
    char *place = (char *) malloc(1000);
    int place_Comment = 0;
    int place_show = 0;
    int count = 0;
    int variable_count = 0;
    char *token;
    //token = &tokens;
    strcpy(token,tokens);

    //printf("%s",token);
    if(token == NULL)
        exit(0);
    else
    {
        
        /*if(strstr(token,"def") || functionTrue == true)
        {
            if(lineCounter == -1)
            {
                news = (chain)malloc(sizeof(func));
                news->newFunc = heads;
                heads = news;
                functionTrue = true;
                lineCounter = 0;
                strcpy(news->funcName,token);
                
                
            }
            else if(lineCounter == 0)
            {
                strcpy(news->lines[lineCounter].line,token);
                // printf("%s \n",news->lines[lineCounter].line);
                lineCounter++;
                
            }
            else if(lineCounter > 0)
            {
                if(strstr(token,"[end]") > 0)
                {
                    strcpy(news->lines[lineCounter].line,token);
                    //  printf("%s \n",news->lines[lineCounter].line);
                    currents = heads;
                    while(currents->newFunc != NULL)
                    {currents = currents->newFunc;}
                    lineCounter = -1;
                    functionTrue = false;
                    
                }
                else
                {
                    strcpy(news->lines[lineCounter].line,token);
                    lineCounter++;
                }
            }
        }
        else if(strstr(token,"()") > 0)
        {
            
            int defcount = 0;
            currents = heads;
            while(currents != NULL)
            {
                if(strstr(currents->funcName,token) > 0)
                {
                    while (!(strstr("[end]",currents->lines[defcount].line) > 0))
                    {
                        
                        char *stringToken = (char *) malloc(1000);
                        strcpy(stringToken,currents->lines[defcount].line);
                        interpreter(stringToken);
                        defcount++;
                    }
                    break;
                }
                else
                {
                    currents = currents->newFunc;
                }
            }
            
        }
        else if(strstr(token,"[if]") > 0 )
        {
            // puts("here 0");
            ifStruct = true;
            newss = (ifSLine)malloc(sizeof(ifStat));
            newss->ifStates = headss;
            headss = newss;
            ifCounter = 1;
            strcpy(newss->ifLine,token);
            strcpy(newss->lines[0].line,token);
            
        }
        else if(ifCounter == 1)
        {
            //puts("here 1");
            strcpy(newss->lines[ifCounter].line,token);
            ifCounter++;
            
        }
        else if(ifCounter > 0)
        {
            // variable to check to make sure structure statment
            // is true
            
            
            if(strstr(token,"[ended]") > 0 || strstr(token,"[ended]\n") > 0)
            {
                
                //puts("here ending");
                strcpy(newss->lines[ifCounter].line,token);
                currentss = headss;
                while(currentss->ifStates != NULL)
                {currentss = currentss->ifStates;}
                ifCounter = -1;
                functionTrue = false;
                
                currentss = headss;
                while(currentss != NULL)
                {
                    
                    if(ifTrue == false)
                    {
                        while(!(strstr("[ended]",currentss->lines[ifLineCount].line) > 0) && !(strstr("[ended]",currentss->lines[ifLineCount].line) > 0))
                        {
                            
                            char *ConditionToken = (char *) malloc(1000);
                            strcpy(ConditionToken,currentss->lines[ifLineCount].line);
                            //printf("%s",ConditionToken);
                            
                            if(strstr(ConditionToken,"[if]") > 0)
                            {
                                float floValue1;
                                float floValue2;
                                char  *strValue1;
                                char  *strValue2;
                                bool boValue1;
                                bool boValue2;
                                bool state;
                                bool valueState =false;
                                int which = -1;
                                bool and,or,not =false;
                                bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                char *ifCondition = (char *) malloc(1000);
                                ifCondition = strtok(ConditionToken," ");
                                
                                while(ifCondition != NULL)
                                {
                                    //printf("%s ",ifCondition);
                                    if(strstr(ifCondition,"[if]") > 0)
                                        ifbool = true;
                                    else if(strstr(ifCondition,"and") > 0)
                                        and = true;
                                    else if(strstr(ifCondition,"or") > 0)
                                        or = true;
                                    else if(strstr(ifCondition,"not") > 0)
                                        not = true;
                                    else if(strstr(ifCondition,"==") > 0)
                                        equal = true;
                                    else if(strstr(ifCondition,">") > 0)
                                        greater = true;
                                    else if(strstr(ifCondition,"<") > 0)
                                        less = true;
                                    else if(strstr(ifCondition,">=") > 0)
                                        greatEqual = true;
                                    else if(strstr(ifCondition,"<=") > 0)
                                        lessEqual = true;
                                    else if(strstr(ifCondition,"!=") > 0)
                                        notEqual = true;
                                    else
                                    {
                                        if(isdigit(ifCondition[0]) && valueState == false)
                                        {
                                            which = 0;
                                            floValue1 = atof(ifCondition);
                                            valueState = true;
                                        }
                                        else if(isdigit(ifCondition[0]) && valueState == true)
                                        {
                                            floValue2 = atof(ifCondition);
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                        {       which = 1;
                                            strcpy(strValue1,ifCondition);
                                            valueState = true;
                                            
                                        }
                                        else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                        {
                                            strcpy(strValue2,ifCondition);
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                        {
                                            which = 2;
                                            boValue1 = true;
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                        {
                                            boValue1 = false;
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                        {
                                            boValue2 = true;
                                            valueState = false;
                                        }
                                        else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                        {
                                            boValue2 = false;
                                            valueState = false;
                                        }
                                        else if(valueState == false)
                                        {
                                            
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(ifCondition,current->name) > 0)
                                                {
                                                    if (isdigit(current->value[0]))
                                                    {
                                                        //printf("%s \n",current->value);
                                                        which = 0;
                                                        floValue1 = atof(current->value);
                                                        break;
                                                    }
                                                    else
                                                    {
                                                        if(strstr(current->value,"true") >0)
                                                        {
                                                            which = 2;
                                                            boValue1 = true;
                                                            break;
                                                        }
                                                        else if(strstr(current->value,"false") >0)
                                                        {
                                                            which = 2;
                                                            boValue1 = false;
                                                            break;
                                                        }
                                                        
                                                        else
                                                        {
                                                            which = 1;
                                                            int size = 0;
                                                            int count = 2;
                                                            int counts = 0;
                                                            size = strlen(current->value);
                                                            while (count <= size-3)
                                                            {
                                                                
                                                                //printf("%c",current->value[count]);
                                                                strValue1[counts] = current->value[count];
                                                                count++;
                                                                counts++;
                                                            }
                                                            //printf("%s",current->value);
                                                            //puts("");
                                                            break;
                                                        }
                                                    }
                                                }
                                                else
                                                    current = current->next;
                                            }
                                            valueState = true;
                                        }
                                        else if(valueState == true)
                                        {
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(ifCondition,current->name) > 0)
                                                {
                                                    if (isdigit(current->value[0]))
                                                    {
                                                        //printf("%s \n",current->value);
                                                        which = 0;
                                                        floValue2 = atof(current->value);
                                                        break;
                                                    }
                                                    else
                                                    {
                                                        if(strstr(current->value,"true") >0)
                                                        {
                                                            which = 2;
                                                            boValue2 = true;
                                                            break;
                                                        }
                                                        else if(strstr(current->value,"false") >0)
                                                        {
                                                            which = 2;
                                                            boValue2 = false;
                                                            break;
                                                        }
                                                        else
                                                        {
                                                            which = 1;
                                                            int size = 0;
                                                            int count = 2;
                                                            int counts = 0;
                                                            size = strlen(current->value);
                                                            while (count <= size-3)
                                                            {
                                                                
                                                                //printf("%c",current->value[count]);
                                                                strValue2[counts] = current->value[count];
                                                                count++;
                                                                counts++;
                                                            }
                                                            //printf("%s",current->value);
                                                            //puts("");
                                                            break;
                                                        }
                                                    }
                                                }
                                                else
                                                    current = current->next;
                                            }
                                            valueState = true;
                                        }
                                    }
                                    
                                    
                                    ifCondition = strtok(NULL," ");
                                }
                                
                                if(which == 0)
                                {
                                    if(equal == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(greater == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 > floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(less == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 < floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(greatEqual == true)
                                    {
                                        if(floValue1 >= floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(lessEqual == true)
                                    {
                                        if(floValue1 <= floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(notEqual == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 != floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    
                                }
                                else if(which == 2)
                                {
                                    if(and == true)
                                    {
                                        if(boValue1 && boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(or == true)
                                    {
                                        if(boValue1 || boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(not == true)
                                    {
                                        if(!boValue1)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(equal == true)
                                    {
                                        if(boValue1 == boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(less == true)
                                    {
                                        if(boValue1 < boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(greater == true)
                                    {
                                        if(boValue1 > boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                }
                                else if(which == 1)
                                {
                                    if(equal == true)
                                    {
                                        if(strValue1 == strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(greater == true)
                                    {
                                        if(strValue1 > strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(less == true)
                                    {
                                        if(strValue1 < strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                }
                                
                                if(ifTrue == true)
                                {
                                    //puts("here");
                                    ifLineCount++;
                                    while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                    {
                                        //printf("%s",currentss->lines[ifLineCount].line);
                                        char ConditionState[1000];
                                        strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                        interpreter(ConditionState);
                                        //interpreter(currentss->lines[ifLineCount].line);
                                        ifLineCount++;
                                        //puts("here");
                                    }
                                    //ifTrue = false;
                                    //break;
                                }
                                else
                                {
                                    ifLineCount++;
                                    while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                        //printf("%s",currentss->lines[ifLineCount].line);
                                        //interpreter(currentss->lines[ifLineCount].line);
                                        ifLineCount++;
                                        
                                    }
                                    ifTrue = false;
                                }
                            }
                            else if(strstr(ConditionToken,"[then if]") > 0)
                            {
                                float floValue1;
                                float floValue2;
                                char  *strValue1;
                                char  *strValue2;
                                bool boValue1;
                                bool boValue2;
                                bool state;
                                bool valueState =false;
                                int which = -1;
                                bool and,or,not =false;
                                bool greater,less,equal,greatEqual,lessEqual,notEqual,ifbool = false;
                                char *ifCondition = (char *) malloc(1000);
                                
                                ifCondition = strtok(ConditionToken," ");
                                
                                while(ifCondition != NULL)
                                {
                                    //printf("%s ",ifCondition);
                                    if(strstr(ifCondition,"[if]") > 0)
                                        ifbool = true;
                                    else if(strstr(ifCondition,"[then") > 0)
                                        ifbool = true;
                                    else if(strstr(ifCondition,"if]") > 0)
                                        ifbool = true;
                                    else if(strstr(ifCondition,"and") > 0)
                                        and = true;
                                    else if(strstr(ifCondition,"or") > 0)
                                        or = true;
                                    else if(strstr(ifCondition,"not") > 0)
                                        not = true;
                                    else if(strstr(ifCondition,"==") > 0)
                                        equal = true;
                                    else if(strstr(ifCondition,">") > 0)
                                        greater = true;
                                    else if(strstr(ifCondition,"<") > 0)
                                        less = true;
                                    else if(strstr(ifCondition,">=") > 0)
                                        greatEqual = true;
                                    else if(strstr(ifCondition,"<=") > 0)
                                        lessEqual = true;
                                    else if(strstr(ifCondition,"!=") > 0)
                                        notEqual = true;
                                    else
                                    {
                                        if(isdigit(ifCondition[0]) && valueState == false)
                                        {
                                            which = 0;
                                            floValue1 = atof(ifCondition);
                                            valueState = true;
                                        }
                                        else if(isdigit(ifCondition[0]) && valueState == true)
                                        {
                                            floValue2 = atof(ifCondition);
                                            valueState = false;
                                        }
                                        else if(strstr(ifCondition,"\"") > 0 && valueState == false)
                                        {       which = 1;
                                            strcpy(strValue1,ifCondition);
                                            valueState = true;
                                            
                                        }
                                        else if(strstr(ifCondition,"\"") > 0 && valueState == true)
                                        {
                                            strcpy(strValue2,ifCondition);
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"true") > 0 && valueState == false)
                                        {
                                            which = 2;
                                            boValue1 = true;
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"false") > 0 && valueState == false)
                                        {
                                            boValue1 = false;
                                            valueState = true;
                                        }
                                        else if(strstr(ifCondition,"true") > 0 && valueState == true)
                                        {
                                            boValue2 = true;
                                            valueState = false;
                                        }
                                        else if(strstr(ifCondition,"false") > 0 && valueState == true)
                                        {
                                            boValue2 = false;
                                            valueState = false;
                                        }
                                        else if(valueState == false)
                                        {
                                            
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(ifCondition,current->name) > 0)
                                                {
                                                    if (isdigit(current->value[0]))
                                                    {
                                                        //printf("%s \n",current->value);
                                                        which = 0;
                                                        floValue1 = atof(current->value);
                                                        break;
                                                    }
                                                    else
                                                    {
                                                        if(strstr(current->value,"true") >0)
                                                        {
                                                            which = 2;
                                                            boValue1 = true;
                                                            break;
                                                        }
                                                        else if(strstr(current->value,"false") >0)
                                                        {
                                                            which = 2;
                                                            boValue1 = false;
                                                            break;
                                                        }
                                                        
                                                        else
                                                        {
                                                            which = 1;
                                                            int size = 0;
                                                            int count = 2;
                                                            int counts = 0;
                                                            size = strlen(current->value);
                                                            while (count <= size-3)
                                                            {
                                                                
                                                                //printf("%c",current->value[count]);
                                                                strValue1[counts] = current->value[count];
                                                                count++;
                                                                counts++;
                                                            }
                                                            //printf("%s",current->value);
                                                            //puts("");
                                                            break;
                                                        }
                                                    }
                                                }
                                                else
                                                    current = current->next;
                                            }
                                            valueState = true;
                                        }
                                        else if(valueState == true)
                                        {
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(ifCondition,current->name) > 0)
                                                {
                                                    if (isdigit(current->value[0]))
                                                    {
                                                        //printf("%s \n",current->value);
                                                        which = 0;
                                                        floValue2 = atof(current->value);
                                                        break;
                                                    }
                                                    else
                                                    {
                                                        if(strstr(current->value,"true") >0)
                                                        {
                                                            which = 2;
                                                            boValue2 = true;
                                                            break;
                                                        }
                                                        else if(strstr(current->value,"false") >0)
                                                        {
                                                            which = 2;
                                                            boValue2 = false;
                                                            break;
                                                        }
                                                        else
                                                        {
                                                            which = 1;
                                                            int size = 0;
                                                            int count = 2;
                                                            int counts = 0;
                                                            size = strlen(current->value);
                                                            while (count <= size-3)
                                                            {
                                                                
                                                                //printf("%c",current->value[count]);
                                                                strValue2[counts] = current->value[count];
                                                                count++;
                                                                counts++;
                                                            }
                                                            //printf("%s",current->value);
                                                            //puts("");
                                                            break;
                                                        }
                                                    }
                                                }
                                                else
                                                    current = current->next;
                                            }
                                            valueState = true;
                                        }
                                    }
                                    
                                    
                                    ifCondition = strtok(NULL," ");
                                }
                                
                                if(which == 0)
                                {
                                    
                                    if(equal == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(greater == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 > floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(less == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 < floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(greatEqual == true)
                                    {
                                        if(floValue1 >= floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(lessEqual == true)
                                    {
                                        if(floValue1 <= floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(notEqual == true)
                                    {
                                        if(floValue1 == floValue2)
                                            ifTrue = false;
                                        else if(floValue1 != floValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                }
                                else if(which == 2)
                                {
                                    if(and == true)
                                    {
                                        if(boValue1 && boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(or == true)
                                    {
                                        if(boValue1 || boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(not == true)
                                    {
                                        if(!boValue1)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(equal == true)
                                    {
                                        if(boValue1 == boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(less == true)
                                    {
                                        if(boValue1 < boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(greater == true)
                                    {
                                        if(boValue1 > boValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                }
                                else if(which == 1)
                                {
                                    if(equal == true)
                                    {
                                        if(strValue1 == strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                    }
                                    else if(greater == true)
                                    {
                                        if(strValue1 > strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    else if(less == true)
                                    {
                                        if(strValue1 < strValue2)
                                            ifTrue = true;
                                        else
                                            ifTrue = false;
                                        
                                    }
                                    
                                }
                                
                                if(ifTrue == true)
                                {
                                    //puts("here");
                                    ifLineCount++;
                                    while (!(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) )
                                    {
                                        //printf("%s",currentss->lines[ifLineCount].line);
                                        char ConditionStates[1000];
                                        strcpy(ConditionStates,currentss->lines[ifLineCount].line);
                                        interpreter(ConditionStates);
                                        ifLineCount++;
                                        //puts("here");
                                    }
                                    //ifTrue = false;
                                    //break;
                                    
                                }
                                else
                                {
                                    ifLineCount++;
                                    while (!(strstr(currentss->lines[ifLineCount].line,"[then if]") > 0) && !(strstr(currentss->lines[ifLineCount].line,"[then]") > 0) ) {
                                        // printf("%s",currentss->lines[ifLineCount].line);
                                        //interpreter(currentss->lines[ifLineCount].line);
                                        ifLineCount++;
                                        
                                    }
                                    ifTrue = false;
                                }
                            }
                            
                            else if(strstr(ConditionToken,"[then]") > 0)
                            {
                                
                                ifTrue = true;
                                
                                
                                if(ifTrue == true)
                                {
                                    ifLineCount++;
                                    while (!(strstr(currentss->lines[ifLineCount].line,"[ended]") > 0)) {
                                        char *ConditionState = (char *) malloc(1000);
                                        strcpy(ConditionState,currentss->lines[ifLineCount].line);
                                        puts("here");
                                        interpreter(ConditionState);
                                        
                                        ifLineCount++;
                                        
                                    }
                                    //ifTrue = false;
                                    break;
                                    
                                }
                                else
                                    ifTrue = true;
                                
                            }
                            if(ifTrue == true)
                                break;
                            else
                                continue;
                            
                            
                            
                        }
                    }
                    else
                        break;
                    
                }
            }
            else
            {
                
                //puts("here added");
                strcpy(newss->lines[ifCounter].line,token);
                ifCounter++;
            }
        }*/
        // show function
       if (strstr(token,"show") > 0)
        {
            
            // printf("%s",token);
            string = strstr(token,"//");
            place = strstr(token,"show");
            place_Comment = string - token;
            place_show = place - token;
            // Dealing with comments
            if(place_Comment < place_show)
                place_Comment = place_Comment;
            else{
                
                // Will do math if it finds an expression
                if(strstr(token,"\"") > 0)
                {
                    //showtok = strtok(token,"\"");
                    showtok = strtok(token,"\"");
                    while (showtok != NULL) {
                        
                        //printf("%s \n",showtok);
                        if (strstr(showtok,"show") > 0)
                            showValue = 1;
                        else
                        {
                            if (strstr(showtok,"\""))
                            {
                                //printf("%s showing >>>>>",showtok);
                                char *stringArray = (char *) malloc(1000);
                                int count =0;
                                while(showtok[count] != '\"')
                                {
                                    stringArray[count] = showtok[count];
                                    count++;
                                }
                                printf("%s ",stringArray);
                                free(stringArray);
                            }
                            else
                                printf("%s ",showtok);
                        }
                        showtok = strtok(NULL,showsplit);
                    }
                    printf("\n");
                }
                else if(strstr(token,"+") > 0  || strstr(token,"-") > 0 || strstr(token,"*") > 0 || strstr(token,"/") > 0)
                {
                    
                    showtok = strtok(token," ");
                    while (showtok != NULL) {
                        
                        if (strstr(showtok,"show") > 0)
                            showValue = 1;
                        else if (strstr(showtok,"+") > 0)
                            add_token = true;
                        else if (strstr(showtok,"-") > 0)
                            sub_token = true;
                        else if (strstr(showtok,"*") > 0)
                            multi_token = true;
                        else if (strstr(showtok,"/") > 0)
                            div_token = true;
                        else if(isnumber(showtok[0]))
                        {
                            if (number_token == 0)
                            {
                                if (multi_token == true)
                                {
                                    number_token = atof(showtok);
                                    sum_token = 1;
                                }
                                else if(div_token == true)
                                {
                                    number_token = atof(showtok);
                                    sum_token = 1;
                                }
                                else
                                    number_token = atof(showtok);
                            }
                            else
                                sum_token = atof(showtok);
                            
                            if(add_token == true)
                            {
                                number_token += sum_token;
                                add_token = false;
                            }
                            else if(sub_token == true)
                            { number_token -= sum_token;
                                sub_token = false;
                            }
                            else if(multi_token == true)
                            {
                                number_token *= sum_token;
                                multi_token = false;
                            }
                            else if(div_token == true)
                            {   number_token /= sum_token;
                                div_token = false;
                            }
                        }
                        showtok = strtok(NULL,showsplit);
                    }
                    // check for decimal place
                    // add that
                    printf("%3.3f \n",number_token);
                    number_token = 0;
                    sum_token = 0;
                    
                }
                else if(strstr(token,"%") > 0)
                {
                    int shnum0 =0;
                    int shnums1 =0;
                    showtok = strtok(token,showsplit);
                    while (showtok != NULL) {
                        // printf("%s \n",showtok );
                        if (strstr(showtok,"%") > 0)
                            //printf("%s \n",showtok );
                            modvalue = 1;
                        else if(strstr(showtok,"show") > 0)
                            modvalue = 1;
                        else if (shnum0 == 0)
                            shnum0 = atoi(showtok);
                        else if (shnum0 > 0 && shnums1 == 0)
                            shnums1 = atoi(showtok);
                        
                        showtok = strtok(NULL," ");
                    }
                    shnum0 %= shnums1;
                    printf("%d \n",shnum0);
                }
                else
                {
                    showtok = strtok(token,showsplit);
                    while (showtok != NULL) {
                        
                        if (strstr(showtok,"show") > 0)
                        {
                            showValue = 1;
                        }
                        else
                        {
                            current = head;
                            while(current != NULL)
                            {
                                if(strstr(showtok,current->name) > 0)
                                {
                                    if (isdigit(current->value[0]))
                                    {
                                        printf("%s \n",current->value);
                                        break;
                                    }
                                    else
                                    {
                                        int size = 0;
                                        int count = 2;
                                        size = strlen(current->value);
                                        while (count <= size-3) {
                                            printf("%c",current->value[count]);
                                            count++;
                                        }
                                        //printf("%s",current->value);
                                        puts("");
                                        break;
                                    }
                                }
                                else
                                    current = current->next;
                            }
                        }
                        showtok = strtok(NULL,showsplit);
                    }
                }
            }
        }
        else if(strstr(token,"//") > 0)
        {
            // continue if // comments
            token = token;
        }
        else if (strstr(token,"sys") > 0)
        {
            // sys commands
            //token = strncpy(token,"   ",3);
            // system(&token[4]);
            
            int sys_value = 0;
            char *sys_token = (char *) malloc(1000);
            showtok = strtok(token," ");
            while (showtok != NULL) {
                
                if (strstr(showtok, "sys") > 0)
                    sys_value = 1;
                else
                {
                    strcat(sys_token,showtok);
                    strcat(sys_token," ");
                    //printf("%s ",sys_token);
                }
                showtok = strtok(NULL,showsplit);
            }
            system(sys_token);
        }
        else if(strstr(token,"[exit]") > 0)
        {
            // Exit function
            exit(0);
        }
        else if(strstr(token,"[r]") > 0)
        {
            
            FILE *readFile;
            char *readingFile  = (char *) malloc(1000);
            int read_value = 0;
            char *read_token;
            char *readLine  = (char *) malloc(1000);
            
            read_token = strtok(token," ");
            while(read_token != NULL)
            {
                if (strstr(read_token,"[r]") > 0)
                {
                    read_value = 1;
                }
                else
                {
                    int length = 0;
                    int count = 0;
                    length = strlen(read_token);
                    
                    while(count < length)
                    {
                        readingFile[count] =  read_token[count];
                        count++;
                    }
                    
                    char lineRead[1000];
                    if( (readFile = fopen(readingFile, "r+")) != NULL)
                    {
                        while(!feof(readFile))
                        {
                            readLine =fgets(lineRead,1000,readFile);
                            if (readLine == NULL)
                                continue;
                            else
                                printf("%s",readLine);
                        }
                    }
                    else if( (readFile = fopen(readingFile, "r+")) == NULL)
                    {
                        printf("%s not found \n",readingFile);
                    }
                }
                read_token = strtok(NULL," ");
            }
            puts("");
        }
        else if(strstr(token,"[w]") > 0)
        {
            FILE  *writeFile;
            char *writingFile = (char *) malloc(1000);
            int write_Value = 0;
            char *write_token;
            char *writeLine = (char *) malloc(1000);;
            char *delimator = "\n";
            write_token = strtok(token," ");
            
            while(write_token != NULL)
            {
                if(strstr(write_token,"[w]") > 0)
                    write_Value = 1;
                else
                {
                    //writingFile = write_token;
                    // printf("%s",write_token);
                    int length = 0;
                    int count = 0;
                    length = strlen(write_token);
                    
                    while(count < length)
                    {
                        writingFile[count] =  write_token[count];
                        count++;
                    }
                    if( (writeFile  = fopen(writingFile, "w+")) != NULL )
                    {
                        while (!strstr(writeLine,delimator)) {
                            
                            fprintf(writeFile,"%s" ,fgets(writeLine,1000,stdin));
                        }
                        
                        fclose(writeFile);
                    }
                    else if ((writeFile  = fopen(writingFile, "w+")) == NULL)
                    {
                        printf("%s not found \n",writingFile);
                    }
                }
                write_token = strtok(NULL," ");
            }
        }
        else if (strstr(token,"[a]") > 0)
        {
            FILE  *appendFile;
            char *appendingFile = (char *) malloc(1000);
            int append_Value = 0;
            char *append_token;
            char *appendLine = (char *) malloc(1000);;
            char *delimator = "\n";
            append_token = strtok(token," ");
            
            while(append_token != NULL)
            {
                if(strstr(append_token,"[a]") > 0)
                    append_Value = 1;
                else
                {
                    //appendingFile = append_token;
                    //printf("%s",append_token);
                    int length = 0;
                    int count = 0;
                    length = strlen(append_token);
                    
                    while(count < length)
                    {
                        appendingFile[count] =  append_token[count];
                        count++;
                    }
                    if( (appendFile  = fopen(appendingFile, "a+")) != NULL )
                    {
                        while (!strstr(appendLine,delimator)) {
                            
                            fprintf(appendFile,"%s" ,fgets(appendLine,1000,stdin));
                        }
                        
                        fclose(appendFile);
                    }
                    else if ((appendFile  = fopen(appendingFile, "a+")) == NULL)
                    {
                        printf("%s not found \n",appendingFile);
                    }
                }
                append_token = strtok(NULL," ");
            }
        }
        else if(strstr(token,"[math]") > 0)
        {
            
            int shnum0 = 0;
            int shnums1 = 0;
            int showValue = 0;
            int addvalue = 0;
            int subvalue = 0;
            int multivalue = 0;
            int divvalue = 0;
            int modvalue = 0;
            int equal;
            int count = 0;
            
            if(strstr(token,"+") > 0  || strstr(token,"-") > 0 || strstr(token,"*") > 0 || strstr(token,"/") > 0)
            {
                showtok = strtok(token," ");
                while (showtok != NULL) {
                    
                    if (strstr(showtok,"[math]") > 0)
                        showValue = 1;
                    else if (strstr(showtok,"+") > 0)
                        add_token = true;
                    else if (strstr(showtok,"-") > 0)
                        sub_token = true;
                    else if (strstr(showtok,"*") > 0)
                        multi_token = true;
                    else if (strstr(showtok,"/") > 0)
                        div_token = true;
                    else if(isnumber(showtok[0]))
                    {
                        if (number_token == 0)
                        {
                            if (multi_token == true)
                            {
                                number_token = atof(showtok);
                                sum_token = 1;
                            }
                            else if(div_token == true)
                            {
                                number_token = atof(showtok);
                                sum_token = 1;
                            }
                            else
                                number_token = atof(showtok);
                        }
                        else
                            sum_token = atof(showtok);
                        
                        if(add_token == true)
                        {
                            number_token += sum_token;
                            add_token = false;
                        }
                        else if(sub_token == true)
                        { number_token -= sum_token;
                            sub_token = false;
                        }
                        else if(multi_token == true)
                        {
                            number_token *= sum_token;
                            multi_token = false;
                        }
                        else if(div_token == true)
                        {   number_token /= sum_token;
                            div_token = false;
                        }
                    }
                    showtok = strtok(NULL,showsplit);
                }
                // check for decimal place
                // add that
                printf("%3.3f \n",number_token);
                number_token = 0;
                sum_token = 0;
            }
            else if(strstr(token,"%") > 0)
            {
                
                showtok = strtok(token,showsplit);
                while (showtok != NULL) {
                    
                    if (strstr(showtok,"%") > 0)
                        //printf("%s \n",showtok );
                        modvalue = 1;
                    else if (shnum0 == 0)
                        shnum0 = atoi(showtok);
                    else if (shnum0 != 0)
                    {shnums1 = atoi(showtok);}
                    showtok = strtok(NULL,showsplit);
                }
                shnum0 %= shnums1;
                printf("%d \n",shnum0);
                shnum0 = 0;
                shnums1 = 0;
            }
            else
            {
                puts("no operator or invalid operator");
                exit(0);
            }
            
        }
        else if (strstr(token,"clear") > 0)
        {
            // clear screen
            system(token);
        }
        else if (strstr(token,"[mkdir]") > 0)
        {
            char *mkdir = (char *) malloc(1000);
            char *mkdir_token;
            int mkdir_value = 0;
            
            mkdir_token = strtok(token," ");
            while (mkdir_token != NULL)
            {
                if(strstr(mkdir_token,"[mkdir]") > 0)
                    mkdir_value = 1;
                else
                {
                    strcat(mkdir,"mkdir ");
                    strcat(mkdir,mkdir_token);
                }
                mkdir_token = strtok(NULL," ");
                
            }
            system(mkdir);
            
        }
        else if(strstr(token,"[remove]") > 0)
        {
            char *rmdir = (char *) malloc(1000);
            char *rmdir_token;
            int rmdir_value = 0;
            
            rmdir_token = strtok(token," ");
            while (rmdir_token != NULL)
            {
                if(strstr(rmdir_token,"[remove]") > 0)
                    rmdir_value = 1;
                else
                {
                    strcat(rmdir,"rmdir ");
                    strcat(rmdir,rmdir_token);
                }
                rmdir_token = strtok(NULL," ");
            }
            
            system(rmdir);
        }
        else if(strstr(token,"[copy]") > 0)
        {
            // copy keyword is for windows
            // find command for windows
            
            char *cp = (char *) malloc(1000);
            char *copy_token = (char *) malloc(1000);
            char *destName = (char *) malloc(1000);
            char *newDestname = (char *) malloc(1000);
            char *dash = (char *) malloc(1000);
            int copy_value = 0;
            int returnOld;
            int returnNew;
            FILE *old,*new;
            int ch;
            int count = 0;
            char *new_token;
            
            new_token = token;
            while (new_token[count] != ' ')
            {
                copy_token[count] = new_token[count];
                count++;
            }
            
            count++;
            int newCount = 0;
            while (new_token[count] != ' ')
            {
                destName[newCount] = new_token[count];
                count++;
                newCount++;
            }
            
            newCount = 0;
            
            while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
            {
                dash[newCount] = new_token[count];
                count++;
                newCount++;
            }
            newCount = 0;
            
            while (new_token[count] != '\n')
            {
                newDestname[newCount] = new_token[count];
                count++;
                newCount++;
            }
            
            if ((old = fopen(destName,"rb")) == NULL )
                returnOld = -1;
            
            if ((new = fopen(newDestname,"wb")) == NULL )
            {
                fclose(old);
                returnNew = -1;
            }
            if (returnOld == -1)
            {
                printf("%s not found \n",destName);
                exit(0);
            }
            else if (returnNew == -1)
            {
                printf("%s not found \n",newDestname);
                exit(0);
            }
            else
            {
                while(1)
                {
                    ch = fgetc(old);
                    
                    if(!feof(old))
                        fputc(ch,new);
                    else
                        break;
                }
                fclose(new);
                fclose(old);
            }
        }
        else if (strstr(token,"[move]") > 0)
        {
            
            char *copy_token = (char *) malloc(1000);
            char *destName = (char *) malloc(1000);
            char *newDestname = (char *) malloc(1000);
            char *dash = (char *) malloc(1000);
            int copy_value = 0;
            int returnOld;
            int returnNew;
            FILE *old,*new;
            int ch;
            int count = 0;
            char *new_token;
            new_token = token;
            
            while (new_token[count] != ' ')
            {
                copy_token[count] = new_token[count];
                count++;
            }
            count++;
            int newCount = 0;
            
            while (new_token[count] != ' ')
            {
                destName[newCount] = new_token[count];
                count++;
                newCount++;
            }
            newCount = 0;
            
            while (!isalnum(new_token[count]) && !(strstr(dash,"-") > 0))
            {
                dash[newCount] = new_token[count];
                count++;
                newCount++;
            }
            newCount = 0;
            
            while (new_token[count] != '\n')
            {
                newDestname[newCount] = new_token[count];
                count++;
                newCount++;
            }
            if ((old = fopen(destName,"rb")) == NULL )
                returnOld = -1;
            
            if ((new = fopen(newDestname,"wb")) == NULL )
            {
                fclose(old);
                returnNew = -1;
            }
            
            if (returnOld == -1)
            {
                printf("%s not found \n",destName);
                exit(0);
            }
            else if (returnNew == -1)
            {
                printf("%s not found \n",newDestname);
                exit(0);
            }
            else
            {
                while(1)
                {
                    ch = fgetc(old);
                    
                    if(!feof(old))
                        fputc(ch,new);
                    else
                        break;
                }
                fclose(new);
                fclose(old);
            }
            remove(destName);
        }
        else if(strstr(token,"+=") > 0)
        {
            char *addTok;
            int addV = 0;
            float addval = 0;
            char *addVar = (char *) malloc(1000);
            char *stringVal = (char *) malloc(1000);
            
            addTok = strtok(token, " ");
            while(addTok != NULL)
            {
                if(strstr(addTok,"+=") > 0)
                {
                    addV = 1;
                }
                else if(addV == 0)
                {
                    current = head;
                    while(current != NULL)
                    {
                        if(strstr(addTok,current->name) > 0)
                        {
                            strcpy(addVar,current->name);
                            addval = atof(current->value);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                else if(addV == 1)
                {
                    addval = addval + atof(addTok);
                    while(current != NULL)
                    {
                        if(strstr(addVar,current->name) > 0)
                        {
                            sprintf(stringVal, "%2.2f", addval);
                            strcpy(current->value,stringVal);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                addTok = strtok(NULL, " ");
            }
        }
        else if(strstr(token,"-=") > 0)
        {
            char *subTok;
            int subV = 0;
            float subval = 0;
            char *subVar = (char *) malloc(1000);
            char *stringVal = (char *) malloc(1000);
            
            subTok = strtok(token, " ");
            while(subTok != NULL)
            {
                if(strstr(subTok,"-=") > 0)
                {
                    subV = 1;
                }
                else if(subV == 0)
                {
                    current = head;
                    while(current != NULL)
                    {
                        if(strstr(subTok,current->name) > 0)
                        {
                            strcpy(subVar,current->name);
                            subval = atof(current->value);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                else if(subV == 1)
                {
                    subval = subval - atof(subTok);
                    while(current != NULL)
                    {
                        if(strstr(subVar,current->name) > 0)
                        {
                            sprintf(stringVal, "%2.2f", subval);
                            strcpy(current->value,stringVal);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                subTok = strtok(NULL, " ");
            }
        }
        else if(strstr(token,"*=") > 0)
        {
            char *mulTok;
            int mulV = 0;
            float mulval = 0;
            char *mulVar = (char *) malloc(1000);
            char *stringVal = (char *) malloc(1000);
            
            mulTok = strtok(token, " ");
            while(mulTok != NULL)
            {
                if(strstr(mulTok,"*=") > 0)
                {
                    mulV = 1;
                }
                else if(mulV == 0)
                {
                    current = head;
                    while(current != NULL)
                    {
                        if(strstr(mulTok,current->name) > 0)
                        {
                            strcpy(mulVar,current->name);
                            mulval = atof(current->value);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                else if(mulV == 1)
                {
                    mulval = mulval * atof(mulTok);
                    while(current != NULL)
                    {
                        if(strstr(mulVar,current->name) > 0)
                        {
                            sprintf(stringVal, "%2.2f", mulval);
                            strcpy(current->value,stringVal);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                mulTok = strtok(NULL, " ");
            }
            
        }
        else if(strstr(token,"/=") > 0)
        {
            char *divTok;
            int divV = 0;
            float divval = 0;
            char *divVar = (char *) malloc(1000);
            char *stringVal = (char *) malloc(1000);
            
            divTok = strtok(token, " ");
            while(divTok != NULL)
            {
                if(strstr(divTok,"/=") > 0)
                {
                    divV = 1;
                }
                else if(divV == 0)
                {
                    current = head;
                    while(current != NULL)
                    {
                        if(strstr(divTok,current->name) > 0)
                        {
                            strcpy(divVar,current->name);
                            divval = atof(current->value);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                else if(divV == 1)
                {
                    divval = divval / atof(divTok);
                    while(current != NULL)
                    {
                        if(strstr(divVar,current->name) > 0)
                        {
                            sprintf(stringVal, "%2.2f", divval);
                            strcpy(current->value,stringVal);
                            break;
                        }
                        else
                            current = current->next;
                    }
                }
                divTok = strtok(NULL, " ");
            }
            
        }
        else if(strstr(token,"=") > 0 )
        {
            char *equals;
            int equalsVal;
            char *eqPLace;
            int places = 0;
            int size = 0;
            int size_tok = 0;
            char *toktok;
            
            if(strstr(token,"\"") > 0)
            {
                new = (link)malloc(sizeof(variables));
                new->next = head;
                head = new;
                toktok = strtok(token," =");
                
                while(toktok != NULL)
                {
                    if(size == 0)
                    {
                        size = strlen(toktok);
                        strcpy(new->name,toktok);
                    }
                    else if (size > 0 )
                    {strcpy(new->value,toktok);}
                    toktok = strtok(NULL, "=");
                }
                size = 0;
                current = head;
                while(current->next != NULL)
                {current = current->next;}
            }
            
                else
                {
                    // allocating for new variable
                    new = (link)malloc(sizeof(variables));
                    new->next = head;
                    head = new;
                    toktok = strtok(token,showsplit);
                    float add,min,mul,div;
                    float val;
                    //
                    while( toktok != NULL ) {
                        
                        if(strstr(toktok,"=") > 0)
                        {equalsVal = 1;}
                        else if(strstr(toktok,"+") > 0)
                        {add = 1;}
                        else if(strstr(toktok,"-") > 0)
                        {min = 1;}
                        else if(strstr(toktok,"*") > 0)
                        {mul = 1;}
                        else if(strstr(toktok,"/") > 0)
                        {div = 1;}
                        else
                        {
                            if (size == 0)
                            {
                                size_tok = strlen(toktok);
                                strcpy(new->name,toktok);
                                size = strlen(new->name);
                            }
                            else if (size > 0)
                            {
                                size_tok = strlen(toktok);
                                char *string = (char *) malloc(1000);
                                int count = 0;
                                
                                if (strstr(toktok,"?") > 0)
                                {
                                    fgets(string,1000,stdin);
                                    strcpy(new->value,string);
                                }
                                else
                                {
                                    if(add == 1)
                                    {
                                        if(isdigit(toktok[0]))
                                        {
                                            add = 0;
                                            val = atof(new->value) + atof(toktok);
                                            char *stringVal = (char *) malloc(1000);
                                            sprintf(stringVal, "%.2f", val);
                                            strcpy(new->value,stringVal);
                                        }
                                        else
                                        {
                                            add = 0;
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(toktok,current->name) > 0)
                                                {
                                                    val = atof(current->value) + atof(new->value);
                                                    char *stringVal = (char *) malloc(1000);
                                                    sprintf(stringVal, "%.2f", val);
                                                    strcpy(new->value,stringVal);
                                                    break;
                                                }
                                                else
                                                    current = current->next;
                                            }
                                        }
                                    }
                                    else if(min == 1)
                                    {
                                        if(isdigit(toktok[0]))
                                        {
                                            min = 0;
                                            val = atof(new->value) - atof(toktok);
                                            char *stringVal = (char *) malloc(1000);
                                            sprintf(stringVal, "%.2f", val);
                                            strcpy(new->value,stringVal);
                                        }
                                        else
                                        {
                                            min = 0;
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(toktok,current->name) > 0)
                                                {
                                                    val = atoi(new->value) - atof(current->value);
                                                    char *stringVal = (char *) malloc(1000);
                                                    sprintf(stringVal, "%.2f", val);
                                                    strcpy(new->value,stringVal);
                                                    break;
                                                }
                                                else
                                                    current = current->next;
                                            }
                                        }
                                    }
                                    else if(mul == 1)
                                    {
                                        if(isdigit(toktok[0]))
                                        {
                                            mul = 0;
                                            val = atof(new->value) * atof(toktok);
                                            char *stringVal = (char *) malloc(1000);
                                            sprintf(stringVal, "%.2f", val);
                                            strcpy(new->value,stringVal);
                                        }
                                        else
                                        {
                                            mul = 0;
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(toktok,current->name) > 0)
                                                {
                                                    val = atoi(new->value) * atof(current->value);
                                                    char *stringVal = (char *) malloc(1000);
                                                    sprintf(stringVal, "%.2f", val);
                                                    strcpy(new->value,stringVal);
                                                    break;
                                                }
                                                else
                                                    current = current->next;
                                            }
                                        }
                                    }
                                    else if(div == 1)
                                    {
                                        if(isdigit(toktok[0]))
                                        {
                                            mul = 0;
                                            val = atof(new->value) / atof(toktok);
                                            char *stringVal = (char *) malloc(1000);
                                            sprintf(stringVal, "%.2f", val);
                                            strcpy(new->value,stringVal);
                                        }
                                        else
                                        {
                                            div = 0;
                                            current = head;
                                            while(current != NULL)
                                            {
                                                if(strstr(toktok,current->name) > 0)
                                                {
                                                    val = atoi(new->value) / atof(current->value);
                                                    char *stringVal = (char *) malloc(1000);
                                                    sprintf(stringVal, "%.2f", val);
                                                    strcpy(new->value,stringVal);
                                                    break;
                                                }
                                                else
                                                    current = current->next;
                                            }
                                        }
                                    }
                                    else if(isdigit(toktok[0]))
                                    {
                                        while(count <= size_tok-1)
                                        {
                                            string[count] = toktok[count];
                                            count++;
                                        }
                                        strcpy(new->value,string);
                                    }
                                    else
                                    {
                                        current = head;
                                        while(current != NULL)
                                        {
                                            if(strstr(toktok,current->name) > 0)
                                            {
                                                strcpy(new->value,current->value);
                                                break;
                                            }
                                            else
                                                current = current->next;
                                        }
                                    }
                                }
                            }
                        }
                        toktok = strtok(NULL, showsplit);
                    }
                    current = head;
                    while(current->next != NULL)
                    {current = current->next;}
                    size = 0;
                }
            }
        else if(strstr(token,"[end]") > 0)
        {
            puts("");
        }
        else
        {
            // Add to statement for
            if(strlen(token) > 1)
            {
                printf("Invalid Statement %s ",token);
                
            }
            else
                //printf("%s",token);
                count = 0;
        }
    }
    
}


