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
VersionInfoCompany=I Go On (Igoon)
VersionInfoDescription=DevLok - Local development environment manager
VersionInfoCopyright=Copyright 2026 I Go On (Igoon)
VersionInfoProductName=DevLok
VersionInfoProductVersion={#NumericVersion}
VersionInfoProductTextVersion={#AppVersion}
AppPublisher=I Go On (Igoon)
AppPublisherURL=https://github.com/Igoon/DevLok
AppSupportURL=https://github.com/Igoon/DevLok/issues
AppUpdatesURL=https://github.com/Igoon/DevLok/releases
AppComments=Local development environment manager
AppCopyright=Copyright 2026 I Go On (Igoon)
LicenseFile=..\LICENSE
InfoBeforeFile=ABOUT.txt
DefaultDirName={localappdata}\Programs\DevLok
DefaultGroupName=DevLok
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
MinVersion=10.0.19045
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\dist
OutputBaseFilename=DevLok-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
UninstallDisplayName=DevLok {#AppVersion}
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
const
  WebView2GUID = '{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}';

function IsWebView2Installed: Boolean;
var
  Version: String;
begin
  Result :=
    RegQueryStringValue(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\' + WebView2GUID, 'pv', Version) or
    RegQueryStringValue(HKLM, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\' + WebView2GUID, 'pv', Version) or
    RegQueryStringValue(HKCU, 'Software\Microsoft\EdgeUpdate\Clients\' + WebView2GUID, 'pv', Version);
end;

function InitializeSetup: Boolean;
var
  ErrorCode: Integer;
begin
  Result := True;
  if not IsWebView2Installed then
  begin
    MsgBox('DevLok memerlukan Microsoft Edge WebView2 Runtime.' + #13#10 + #13#10 +
      'Installer akan membuka halaman unduh resmi Microsoft. Jalankan DevLok Setup lagi setelah WebView2 terpasang.',
      mbError, MB_OK);
    ShellExec('open', 'https://go.microsoft.com/fwlink/p/?LinkId=2124703', '', '', SW_SHOWNORMAL, ewNoWait, ErrorCode);
    Result := False;
    Exit;
  end;
  if not FileExists(ExpandConstant('{sys}\msvcp140.dll')) then
  begin
    MsgBox('Visual C++ Runtime 2015-2022 x64 belum terdeteksi. Pasang Microsoft Visual C++ Redistributable x64 sebelum menjalankan PHP/MySQL.',
      mbInformation, MB_OK);
  end;
end;

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
  CopyFile(Current + '\devlok-gui.exe', Dir + '\devlok-gui.exe', False);
  CopyFile(Current + '\dl.exe', Dir + '\dl.exe', False);
  CopyFile(Current + '\dl-helper.exe', Dir + '\dl-helper.exe', False);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssInstall then
    BackupBinaries;
end;
