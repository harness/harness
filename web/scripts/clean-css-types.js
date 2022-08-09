/* eslint-disable @typescript-eslint/no-var-requires, no-console */

/**
 * Since all the ".css.d.ts" files are generated automatically from webpack, developers might be
 * aware of its existence.
 *
 * Example: "MyAwesomeComponent.css" file will have a "MyAwesomeComponent.css.d.ts" file
 *
 * When a ".css" file is deleted, the corresponding ".css.d.ts" file must be deleted too, but this is
 * hard to do in a fast-paced development environment.
 *
 * The rationale here is that, since these files are generated automatically, they must be cleaned
 * automatically too.
 *
 * How do we do it?
 * We glob for all the ".css.d.ts" files and check if it has a corresponding ".css" file
 * If it doesn't, we delete that ".css.d.ts" file
 */

const fs = require('fs');
const glob = require('glob');

const files = glob.sync('src/**/*.css.d.ts');
console.log(`Found ${files.length} '.css.d.ts' files`);

let i = 0;

files.forEach(file => {
  // for every '.css' there will be a coresponding '.css.d.ts' file and vice versa
  const cssFile = file.replace('.d.ts', '');

  if (!fs.existsSync(cssFile)) {
    console.log(`Deleting "${file}" because corresponding "${cssFile}" does not exist`);
    fs.unlinkSync(file);
    i++;
  }
});

console.log(`Deleted total of ${i} '.css.d.ts' files`);
