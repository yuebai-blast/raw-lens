export interface Summary {
  id: string
  time: string
  remoteAddr: string
  tls: boolean
  method: string
  target: string
  proto: string
  name: string
  headerCount: number
  bodySize: number
  rawSize: number
}

export interface Header {
  name: string
  value: string
}

export interface Detail extends Summary {
  requestLine: string
  headers: Header[]
  rawBase64: string
  bodyBase64: string
}

export interface SessionInfo {
  enabled: boolean
  authenticated: boolean
}

export interface Meta {
  captureUrl: string
}
