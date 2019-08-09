#!/bin/sh -e

file="$1"

[ -z "$file" ] && echo "please provide filename" && exit 1

redress=$(mktemp -t byebyegocheck.redress.XXXXXXXX.go)
hash=$(mktemp -t byebyegocheck.sha.XXXXXXXX)

clean() {
	rm -rf "$redress" "$hash"
}

trap clean EXIT

cat > "$redress" <<EOF
// +build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

func main() {
	pattern := os.Args[1]
	file := os.Args[2]

	err := run(pattern, file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(pattern string, file string) error {
	rgx, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpName := file + ".tmp"
	fixed, err := os.Create(tmpName)
	if err != nil {
		return err
	}
	defer fixed.Close()

	found := false
	s := bufio.NewScanner(f)
	for s.Scan() {
		b := s.Bytes()
				fixed.Write(b)
		if !found {
			if !rgx.Match(b) {
				fixed.Write([]byte{'\n'})
			} else {
				found = true
			}
			continue
		}
		if len(b) == 0 || b[len(b)-1] != ',' {
			fixed.Write([]byte{'\n'})
			found = false
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	fixed.Close()
	f.Close()
	return os.Rename(tmpName, file)
}
EOF

cmp=$(grep -m 1 '"gotest.tools/assert/cmp"' "$file" | awk '{print $1}')
if [ -z "$cmp" ]; then
	cmp=cmp
fi

run() {
	shasum -a 256 "$file" > "$hash"

	# redress multiline Assert calls
	go run "$redress" '\bc\.Assert\(.*,$' "$file"
	
	# check.C -> testing.T
	sed -E -i'' 's,\bcheck\.C\b,testing.T,g' "$file"
	# c -> t in definitions/declarations
	#sed -E -i'' 's,\bc \*testing\.T\b,t *testing.T,g' "$file"
	# c -> t for all methods that are supported by testing.T
	#methods=$(go doc testing.T | grep '*T) ' | cut -d' ' -f4 | cut -d'(' -f1)
	#methodsRegex=$(echo $methods | tr ' ' '|')
	#sed -E -i'' "s,\bc\.($methodsRegex)\(,t.\1(,g" "$file"
	
	# normalize to checker
	sed -E -i'' 's,\bcheck\.(Equals|DeepEquals|HasLen|IsNil|Matches|Not|NotNil)\b,checker.\1,g' "$file"
	
	# handle check.Commentf
	sed -E -i'' 's#\bcheck.Commentf\(([^,]+),(.*)\)#\nfmt.Sprintf(\1,\2)#g' "$file"
	sed -E -i'' 's#\bcheck.Commentf\(([^\)]+)\)#\n\1#g' "$file"
	sed -E -i'' 's#\bcheck.Commentf\(("[^"]+")\)#\n\1#g' "$file"
	
	# handle Not(IsNil)
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Not\(checker\.IsNil\)#assert.Assert(c, \1 != nil#g" "$file"
	# handle Not(Equals)
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Not\(checker\.Equals\), (.*)#assert.Assert(c, \1 != \2#g" "$file"
	# handle Not(Contains)
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Not\(checker\.Contains\), (.*), *$#assert.Assert(c, !strings.Contains(\1, \2),#g' "$file"
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Not\(checker\.Contains\), (.*)#assert.Assert(c, !strings.Contains(\1, \2)#g' "$file"
	# handle Not(Matches)
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Not\(checker\.Matches\), (.*), *\$#assert.Assert(c, !${cmp}.Regexp(\"^\"+\2+\"\$\", \1)().Success(),#g" "$file"
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Not\(checker\.Matches\), (.*)\)#assert.Assert(c, !${cmp}.Regexp(\"^\"+\2+\"\$\", \1)().Success())#g" "$file"
	
	#grep -nE '\bchecker\.Not\(' "$file" && echo "ERROR: Found unhandled check.Not instances" && exit 1
	
	# Equals
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Equals, (.*), *$#assert.Equal(c, \1, \2,#g' "$file"
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Equals, (.*)#assert.Equal(c, \1, \2#g' "$file"
	# DeepEquals
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.DeepEquals, (.*), *$#assert.DeepEqual(c, \1, \2,#g' "$file"
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.DeepEquals, (.*)#assert.DeepEqual(c, \1, \2#g' "$file"
	# HasLen
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.HasLen, (.*), *$#assert.Equal(c, len(\1), \2,#g' "$file"
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.HasLen, (.*)#assert.Equal(c, len(\1), \2#g' "$file"
	# IsNil
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.IsNil#assert.Assert(c, \1 == nil#g' "$file"
	# NotNil
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.NotNil#assert.Assert(c, \1 != nil#g' "$file"
	# Matches
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Matches, (.*), *\$#assert.Assert(c, ${cmp}.Regexp(\"^\"+\2+\"\$\", \1),#g" "$file"
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.Matches, (.*)\)#assert.Assert(c, ${cmp}.Regexp(\"^\"+\2+\"\$\", \1))#g" "$file"
	# Contains
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Contains, (.*), *$#assert.Assert(c, strings.Contains(\1, \2),#g' "$file"
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.Contains, (.*)#assert.Assert(c, strings.Contains(\1, \2)#g' "$file"
	# GreaterThan
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.GreaterThan, (.*), *\$#assert.Assert(c, \1 > \2,#g" "$file"
	sed -E -i'' "s#\bc\.Assert\((.*), checker\.GreaterThan, (.*)#assert.Assert(c, \1 > \2#g" "$file"
	# False
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.False#assert.Assert(c, !\1#g' "$file"
	# True
	sed -E -i'' 's#\bc\.Assert\((.*), checker\.True#assert.Assert(c, \1#g' "$file"
	
	
	# c, -> t,
	#sed -E -i'' 's#\bc,#t,#g' "$file"
	# (c) -> (t),
	#sed -E -i'' 's#\(c\)#(t)#g' "$file"

	# redress check.Suite calls
	go run "$redress" '[^/]\bcheck\.Suite\(.*\{$' "$file"

	go run "$redress" '\bassert\..*,$' "$file"

	goimports -w "$file"
	gofmt -w -s "$file"

	# comment out check.Suite calls
	sed -E -i'' 's#([^/])(check\.Suite\()#\1//\2#g' "$file"
	# comment out check.TestingT
	sed -E -i'' 's#([^/])(check\.TestingT\()#\1//\2#g' "$file"
}

n=1
while :; do
	echo round $n
	run
	n=$((n + 1))
	if shasum -s -c "$hash" -a 256; then
		exit 0
	fi
done
