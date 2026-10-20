#ifndef AppVersion
  #define AppVersion "0.1.0"
#endif
#ifndef NumericVersion
  #define NumericVersion "0.1.0"
#endif
#ifndef DistDir
  #define DistDir "..\dist\DevLok\0.1.0-dev"
#endif

[Setup]
AppId={{A64B6BBC-CB5F-4D3B-9B40-5088A1C23FE0}
AppName=DevLok
AppVersion={#AppVersion}
AppVerName=DevLok {#AppVersion}
VersionInfoVersion={#NumericVersion}
AppPublisher=DevLok
DefaultDirName={localappdata}\Programs\DevLok
DefaultGroupName=DevLok
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\dist
OutputBaseFilename=DevLok-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
UninstallDisplayName=DevLok
UninstallDisplayIcon={app}\devlok-gui.exe
SetupIconFile=DevLok.ico

[Tasks]
Name: "desktopicon"; Description: "Buat shortcut Desktop"; GroupDescription: "Shortcut:"

[Files]
Source: "{#DistDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\DevLok"; Filename: "{app}\devlok-gui.exe"
Name: "{group}\Uninstall DevLok"; Filename: "{uninstallexe}"
Name: "{autodesktop}\DevLok"; Filename: "{app}\devlok-gui.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\devlok-gui.exe"; Description: "Jalankan DevLok"; Flags: nowait postinstall skipifsilent

[Code]
procedure BackupBinaries;
var
  Dir: String;
  Current: String;
begin
  Current := ExpandConstant('{app}');
  if not FileExists(Current + '\devlok-gui.exe') then
    Exit;
  Dir := Current + '\backup\previous';
  ForceDirectories(Dir);
  FileCopy(Current + '\devlok-gui.exe', Dir + '\devlok-gui.exe', False);
  FileCopy(Current + '\dl.exe', Dir + '\dl.exe', False);
  FileCopy(Current + '\dl-helper.exe', Dir + '\dl-helper.exe', False);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssInstall then
    BackupBinaries;
end;
