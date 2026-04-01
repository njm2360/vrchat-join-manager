#define MyAppName "VRChat Join Manager Agent"
#define MyAppVersion "v1.0.0"
#define MyAppExeName "vjm-agent.exe"

[Setup]
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName}
AppPublisher=njm2360
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputDir=dist
OutputBaseFilename=Setup
Compression=lzma
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
UsedUserAreasWarning=no

[Languages]
Name: "japanese"; MessagesFile: "compiler:Languages\Japanese.isl"

[Files]
Source: "{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Dirs]
Name: "{localappdata}\{#MyAppName}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Parameters: "-install"; Flags: runhidden waituntilterminated

[UninstallRun]
Filename: "{app}\{#MyAppExeName}"; Parameters: "-remove"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveSvc"

[Code]
var
  SettingsPage: TInputQueryWizardPage;
  LogDirPage: TInputDirWizardPage;

procedure InitializeWizard;
var
  VRChatPath: String;
begin
  SettingsPage := CreateInputQueryPage(wpSelectDir,
    '接続設定', 'サーバーへの接続情報を入力してください', '');
  SettingsPage.Add('API Base URL (例: http://192.168.1.1:8080):', False);
  SettingsPage.Add('タイムゾーン (例: Asia/Tokyo):', False);

  SettingsPage.Values[0] := '';
  SettingsPage.Values[1] := 'Asia/Tokyo';

  LogDirPage := CreateInputDirPage(SettingsPage.ID,
    'ログディレクトリ', 'VRChatのログフォルダを選択してください',
    'VRChatのログが保存されているフォルダを指定してください。',
    False, '');
  LogDirPage.Add('ログディレクトリ');

  VRChatPath := GetEnv('USERPROFILE') + '\AppData\LocalLow\VRChat\VRChat';
  if DirExists(VRChatPath) then
    LogDirPage.Values[0] := VRChatPath;
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if CurPageID = SettingsPage.ID then begin
    if Trim(SettingsPage.Values[0]) = '' then begin
      MsgBox('API Base URLを入力してください。', mbError, MB_OK);
      Result := False;
      Exit;
    end;
    if Trim(SettingsPage.Values[1]) = '' then begin
      MsgBox('タイムゾーンを入力してください。', mbError, MB_OK);
      Result := False;
      Exit;
    end;
  end;
  if CurPageID = LogDirPage.ID then begin
    if Trim(LogDirPage.Values[0]) = '' then begin
      MsgBox('ログディレクトリを選択してください。', mbError, MB_OK);
      Result := False;
      Exit;
    end;
    if not DirExists(LogDirPage.Values[0]) then begin
      MsgBox('指定されたディレクトリが存在しません。', mbError, MB_OK);
      Result := False;
      Exit;
    end;
  end;
end;

procedure WriteEnvFile;
var
  EnvPath: String;
  Lines: TArrayOfString;
begin
  EnvPath := ExpandConstant('{localappdata}\{#MyAppName}\.env');
  SetArrayLength(Lines, 3);
  Lines[0] := 'API_BASE=' + Trim(SettingsPage.Values[0]);
  Lines[1] := 'LOG_TZ=' + Trim(SettingsPage.Values[1]);
  Lines[2] := 'LOG_DIR=' + Trim(LogDirPage.Values[0]);
  SaveStringsToFile(EnvPath, Lines, False);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    WriteEnvFile;
end;
