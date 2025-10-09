# Copyright 2024 Harness, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# this is needed because "When input files are specified on the command line, tsconfig.json files are ignored."
# so this is the only way to run tsc with filenames and tsconfig together :'(
# see "https://www.typescriptlang.org/docs/handbook/tsconfig-json.html"

files="";

# lint-staged will pass all files in $1 $2 $3 etc. iterate and concat.
for var in "$@"
do
    files="$files \"$var\","
done

# create temporary tsconfig which includes only passed files
str="{
  \"extends\": \"./tsconfig.json\",
  \"include\": [\"src/global.d.ts\", \"src/ar/global.d.ts\", $files]
}"
echo $str > tsconfig.tmp

# run typecheck using temp config
tsc -p ./tsconfig.tmp

# capture exit code of tsc
code=$?

# delete temp config
rm ./tsconfig.tmp

exit $code
