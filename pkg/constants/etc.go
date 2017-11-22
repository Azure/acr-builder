package constants

import "time"

// ObfuscationString is the string is used to hide sensitive data such as token in logs
const ObfuscationString = "*************"

// TimestampFormat is the common timestamp format ACR Builder uses
const TimestampFormat = time.RFC3339

// DefaultDockerfile is the name of the default dockerfile
const DefaultDockerfile = "Dockerfile"

// SourceNameWebArchive is the name of the web archive source
const SourceNameWebArchive = "web archive"

// SourceNameGit is the name of the git source
const SourceNameGit = "git repository"

// SourceNameLocal is the name of local source
const SourceNameLocal = "local directory"
