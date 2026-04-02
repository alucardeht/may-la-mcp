package lang

import "regexp"

var funcPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s+)?([a-zA-Z_]\w*)\s*\(`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z_$]\w*)\s*\(`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([a-zA-Z_$]\w*)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[a-zA-Z_$]\w*)\s*=>`),
	regexp.MustCompile(`^\s*(?:async\s+)?def\s+([a-zA-Z_]\w*)\s*\(`),
	regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+([a-zA-Z_]\w*)\s*[<(]`),
	regexp.MustCompile(`^\s*(?:public|private|protected|static|virtual|override|inline)?\s*\w+\s+([a-zA-Z_]\w*)\s*\(`),
	regexp.MustCompile(`^\s*def\s+([a-zA-Z_]\w*[!?]?)`),
}
