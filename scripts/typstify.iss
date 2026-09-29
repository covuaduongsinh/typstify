; Inno Setup template. Duoc goi boi scripts/package-desktop.ps1 voi
; /DAppVersion /DStageDir /DOutDir /DChessbookVersion /DBoardVersion.
[Setup]
AppId={{6B0D2C1E-7A54-4E7B-9C1B-7F5E3D2A9B11}
AppName=Typstify
AppVersion={#AppVersion}
AppPublisher=Duong Sinh
DefaultDirName={localappdata}\Programs\Typstify
DefaultGroupName=Typstify
PrivilegesRequired=lowest
OutputDir={#OutDir}
OutputBaseFilename=Typstify-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64compatible
ArchitecturesAllowed=x64compatible
UninstallDisplayIcon={app}\Typstify.exe
DisableProgramGroupPage=yes

[Tasks]
Name: "desktopicon"; Description: "Tao bieu tuong ngoai Desktop"; Flags: unchecked

[Files]
Source: "{#StageDir}\Typstify.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#StageDir}\bin\*"; DestDir: "{app}\bin"; Flags: ignoreversion recursesubdirs
Source: "{#StageDir}\chessbook-lib\*"; DestDir: "{userappdata}\typst\packages\local\chessbook\{#ChessbookVersion}"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#StageDir}\board-n-pieces\*"; DestDir: "{userappdata}\typst\packages\preview\board-n-pieces\{#BoardVersion}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Typstify"; Filename: "{app}\Typstify.exe"
Name: "{userdesktop}\Typstify"; Filename: "{app}\Typstify.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\Typstify.exe"; Description: "Chay Typstify"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\typst\packages\local\chessbook\{#ChessbookVersion}"
