declare global {
  interface Window {
    runtime?: {
      EventsOn(event: string, callback: (...data: any[]) => void): void
      EventsOff(event: string, ...additionalEvents: string[]): void
    }
    go?: {
      main?: {
        App?: {
          Status(): Promise<any>
          Up(path: string): Promise<any>
          Down(): Promise<void>
          Sites(): Promise<any[]>
          CreateSite(name: string, php: string, withDatabase: boolean): Promise<any>
          Components(path: string): Promise<any>
          MySQLVersions(): Promise<string[]>
          SetMySQLVersion(path: string, version: string): Promise<void>
          ServiceVersions(service: string): Promise<string[]>
          ServiceVersionOptions(service: string): Promise<{ version: string; channel?: string; recommended?: boolean }[]>
          CheckPorts(ports: number[]): Promise<any[]>
          ScanLaragon(path: string): Promise<any[]>
          ImportLaragon(path: string, names: string[], adopt: boolean): Promise<any[]>
          SaveProjectConfig(path: string, input: any): Promise<void>
          SetComponents(path: string, database: boolean, phpmyadmin: boolean): Promise<void>
          OpenURL(url: string): Promise<void>
          OpenFolder(path: string): Promise<void>
          OpenEditor(path: string): Promise<void>
          OpenTerminal(path: string): Promise<void>
          RuntimeList(): Promise<any[]>
          RemoveRuntime(name: string, version: string): Promise<string>
          RuntimeCache(): Promise<any>
          CleanupRuntimeCache(): Promise<string>
          CleanupUnusedRuntimes(): Promise<string>
          InstallRuntimeUpdate(name: string, version: string): Promise<string>
          UpdateAllPatches(): Promise<string>
          CancelUpdate(): Promise<void>
          UpdatePreferences(): Promise<any>
          SaveUpdatePreferences(prefs: any): Promise<void>
          Logs(service: string, lines: number): Promise<string>
          Version(): Promise<string>
        }
      }
    }
  }
}

export {}
