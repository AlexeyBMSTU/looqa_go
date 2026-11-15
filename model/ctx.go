package model

type CtxKey string

const RequestIDKey CtxKey = "requestID"

// AuthClaimsKey — ключ под claims access-токена, положенные в контекст
// мидлварёй аутентификации (looqa/internal/auth.Claims).
const AuthClaimsKey CtxKey = "authClaims"
