export type RiskCaseUser = {
  case_id: number
  user_id: number
  username: string
  score: number
  evidence_json: string
  banned_by_case: boolean
}

export type RiskCase = {
  id: number
  status: 'open' | 'ignored' | 'banned' | 'acknowledged'
  score: number
  categories: number
  reason_summary: string
  first_seen: number
  last_seen: number
  created_at: number
  reviewed_by: number
  reviewed_at: number
  resolution: string
  users: RiskCaseUser[]
}

export type RiskCasePage = { items: RiskCase[]; total: number }
