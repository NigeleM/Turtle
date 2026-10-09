; The Windows installer: turtle-windows-amd64-setup.exe.
;
; Installs turtle.exe for all users in Program Files and puts it on the
; system PATH; makes .turtle Turtle's file type (double-click runs the
; script in its own folder and waits for a key; right-click has Run with
; Turtle and Edit);
; adds Turtle to "Open with" for .trt, the default only where no other
; program has it (.trt is shared with AvaSoft and TensorRT files); and adds
; a Start menu entry for the Turtle prompt. Uninstalling takes all of it
; back, leaving any other program's file types alone. Installing a newer
; version over an older one upgrades it in place.
;
; Built by the release (release.yml) with Inno Setup 6:
;
;   iscc /DAppVersion=0.9.170 /DTurtleExe=..\release\turtle-windows-amd64.exe installer\turtle.iss
;
; IconFile is the icon for the installer, .turtle files, the Start menu
; and Apps & features: the test icon until the final logo is ready.

#ifndef IconFile
  #define IconFile "..\test-icons\shell-one.ico"
#endif
#define HasIcon FileExists(AddBackslash(SourcePath) + IconFile)
#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
#ifndef TurtleExe
  #define TurtleExe "..\turtle.exe"
#endif
#ifndef OutputDir
  #define OutputDir "..\release"
#endif

#define ProgId "Turtle.Script"

[Setup]
; The same AppId on every version is what makes a newer one an upgrade.
AppId={{C4393E4C-B33C-48E5-95C6-EBA8F5F53EF2}
AppName=Turtle
AppVersion={#AppVersion}
AppVerName=Turtle {#AppVersion}
AppPublisher=Nigele McCoy
AppPublisherURL=https://github.com/NigeleM/Turtle
AppSupportURL=https://github.com/NigeleM/Turtle/issues
AppUpdatesURL=https://github.com/NigeleM/Turtle/releases
DefaultDirName={autopf}\Turtle
DisableProgramGroupPage=yes
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
; Windows 10 and later: the prompt's colors need them.
MinVersion=10.0
ChangesEnvironment=yes
ChangesAssociations=yes
CloseApplications=yes
#if HasIcon
UninstallDisplayIcon={app}\turtle.ico
#else
UninstallDisplayIcon={app}\turtle.exe
#endif
UninstallDisplayName=Turtle
OutputDir={#OutputDir}
OutputBaseFilename=turtle-windows-amd64-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
#if HasIcon
SetupIconFile={#IconFile}
#endif

[Files]
Source: "{#TurtleExe}"; DestDir: "{app}"; DestName: "turtle.exe"; Flags: ignoreversion
Source: "turtle-run.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion
Source: "..\NOTICE"; DestDir: "{app}"; DestName: "NOTICE.txt"; Flags: ignoreversion
#if HasIcon
Source: "{#IconFile}"; DestDir: "{app}"; DestName: "turtle.ico"; Flags: ignoreversion
#endif

[Icons]
; The prompt opens in the user's own folder, whoever runs it.
#if HasIcon
Name: "{autoprograms}\Turtle\Turtle"; Filename: "{app}\turtle.exe"; WorkingDir: "%USERPROFILE%"; IconFilename: "{app}\turtle.ico"; Comment: "The Turtle prompt"
#else
Name: "{autoprograms}\Turtle\Turtle"; Filename: "{app}\turtle.exe"; WorkingDir: "%USERPROFILE%"; Comment: "The Turtle prompt"
#endif
Name: "{autoprograms}\Turtle\Turtle documentation"; Filename: "https://github.com/NigeleM/Turtle/tree/main/docs"

[Registry]
; The file type. "turtlerun" is the same verb as the right-click entry
; below, so Explorer shows it once.
Root: HKLM; Subkey: "Software\Classes\{#ProgId}"; ValueType: string; ValueData: "Turtle script"; Flags: uninsdeletekey
#if HasIcon
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\DefaultIcon"; ValueType: string; ValueData: "{app}\turtle.ico"
#endif
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell"; ValueType: string; ValueData: "turtlerun"
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell\turtlerun"; ValueType: string; ValueData: "Run with Turtle"
#if HasIcon
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell\turtlerun"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\turtle.ico"
#endif
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell\turtlerun\command"; ValueType: string; ValueData: """{app}\turtle-run.cmd"" ""%1"" %*"
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell\edit"; ValueType: string; ValueData: "Edit"
Root: HKLM; Subkey: "Software\Classes\{#ProgId}\shell\edit\command"; ValueType: string; ValueData: """{win}\notepad.exe"" ""%1"""
; "Open with" for both endings. Which program is the default is set in
; [Code], so it's only taken back if it's still Turtle.
Root: HKLM; Subkey: "Software\Classes\.turtle\OpenWithProgids"; ValueType: string; ValueName: "{#ProgId}"; ValueData: ""; Flags: uninsdeletevalue uninsdeletekeyifempty
Root: HKLM; Subkey: "Software\Classes\.trt\OpenWithProgids"; ValueType: string; ValueName: "{#ProgId}"; ValueData: ""; Flags: uninsdeletevalue uninsdeletekeyifempty
; Run with Turtle on right-click, whatever program opens .turtle files.
Root: HKLM; Subkey: "Software\Classes\SystemFileAssociations\.turtle\shell\turtlerun"; ValueType: string; ValueData: "Run with Turtle"; Flags: uninsdeletekey
#if HasIcon
Root: HKLM; Subkey: "Software\Classes\SystemFileAssociations\.turtle\shell\turtlerun"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\turtle.ico"
#endif
Root: HKLM; Subkey: "Software\Classes\SystemFileAssociations\.turtle\shell\turtlerun\command"; ValueType: string; ValueData: """{app}\turtle-run.cmd"" ""%1"" %*"

[Code]
const
  EnvKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';
  ProgId = '{#ProgId}';

{ ---- PATH ---- }

function InPath(Paths, Dir: string): Integer;
begin
  Result := Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(Paths) + ';');
end;

procedure AddToPath(Dir: string);
var
  Paths: string;
begin
  if not RegQueryStringValue(HKLM, EnvKey, 'Path', Paths) then
    Paths := '';
  if InPath(Paths, Dir) > 0 then
    exit;
  if (Paths <> '') and (Paths[Length(Paths)] <> ';') then
    Paths := Paths + ';';
  RegWriteExpandStringValue(HKLM, EnvKey, 'Path', Paths + Dir);
end;

procedure RemoveFromPath(Dir: string);
var
  Paths: string;
  P: Integer;
begin
  if not RegQueryStringValue(HKLM, EnvKey, 'Path', Paths) then
    exit;
  P := InPath(Paths, Dir);
  if P = 0 then
    exit;
  Paths := ';' + Paths + ';';
  Delete(Paths, P, Length(Dir) + 1);
  RegWriteExpandStringValue(HKLM, EnvKey, 'Path', Copy(Paths, 2, Length(Paths) - 2));
end;

{ ---- which program opens .turtle and .trt ---- }

function DefaultFor(Ext: string): string;
begin
  if not RegQueryStringValue(HKCR, Ext, '', Result) then
    Result := '';
end;

{ Turtle opens Ext: always for .turtle; for .trt only if nothing else does. }
procedure Claim(Ext: string; Always: Boolean);
begin
  if Always or (DefaultFor(Ext) = '') then
    RegWriteStringValue(HKLM, 'Software\Classes\' + Ext, '', ProgId);
end;

{ Takes Turtle back off Ext, if it's still the one opening it. }
procedure Release(Ext: string);
var
  Current: string;
begin
  if RegQueryStringValue(HKLM, 'Software\Classes\' + Ext, '', Current) and (Current = ProgId) then
    RegDeleteValue(HKLM, 'Software\Classes\' + Ext, '');
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    AddToPath(ExpandConstant('{app}'));
    Claim('.turtle', True);
    Claim('.trt', False);
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
  begin
    RemoveFromPath(ExpandConstant('{app}'));
    Release('.turtle');
    Release('.trt');
  end;
  if CurUninstallStep = usPostUninstall then
  begin
    RegDeleteKeyIfEmpty(HKLM, 'Software\Classes\.turtle');
    RegDeleteKeyIfEmpty(HKLM, 'Software\Classes\.trt');
    RegDeleteKeyIfEmpty(HKLM, 'Software\Classes\SystemFileAssociations\.turtle\shell');
    RegDeleteKeyIfEmpty(HKLM, 'Software\Classes\SystemFileAssociations\.turtle');
  end;
end;
