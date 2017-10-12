package constants

import "time"

// ObfuscationString is the string is used to hide sensitive data such as token in logs
const ObfuscationString = "*************"

// TimestampFormat is the common timestamp format ACR Builder uses
const TimestampFormat = time.RFC3339
