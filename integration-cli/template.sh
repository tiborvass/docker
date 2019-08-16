#!/bin/sh -ex

eg -w -t eg_waitAndAssert1.template . 2>&1 | tee /tmp/result
for file in $(awk '{print $2}' /tmp/result); do
	# removing vendor/ in import paths, not sure why eg adds them
	sed -E -i'' 's#^([ \t]+").*/vendor/([^"]+)#\1\2#g' "$file"
	sed -E -i'' 's#\.\(_EG_compareFunc\)##g' "$file"
	goimports -w "$file"
done
