package constants

const TokenError = "could not decode  token %s got error %s "
const AuthorizationFailed = "authorization  failed"
const InternalServerError = "Internal server error"
const DbError = "DB got error %s"
const DateParseError = "StringToTime error converting %s to time %s"
const DataBindingError = "Error binding data %s "

const DESCRIPTION = "description"
const DATA = "data"

// Constants for menu prefixes
const (
	RESPONSE_TYPE_CON = "CON" // Continues the session
	RESPONSE_TYPE_END = "END" // Ends the session
)
