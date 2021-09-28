#!/usr/bin/env node

// Clearly inspired by https://github.com/hfour/docsify-tools/blob/master/src/docsify-auto-sidebar.ts

import * as fs from 'fs';
import * as path from 'path';
import * as yargs from 'yargs';

let ignores = /node_modules|^\.|_sidebar|_docsify|_navbar|_toc.md/;
let isDoc = /.md$/;

type Entry = {
  name: string | null;
  fileName: string;
  link: string;
  children?: Entry[];
};

function niceName (name: string) {
  let splitName = name.split('-');
  if (Number.isNaN(Number(splitName[0]))) {
    return splitName.join(' ');
  }
  return splitName.slice(1).join(' ');
}

function extractTitle (entryPath: string): string {
  // Load file and extract the first header we find
  const readmeFile = fs.statSync(entryPath).isDirectory() ? path.join(entryPath, 'README.md') : entryPath;
  if (fs.existsSync(readmeFile)) {
    const content = fs.readFileSync(readmeFile, {encoding: 'utf-8'});
    const firstHeaderMatch = content.match(/^#+\s+(.+)$/im);
    if (firstHeaderMatch) {
      return firstHeaderMatch[1];
    }
  }
  return entryPath;
}

function buildTree (dirPath: string, name = '', dirLink = ''): Entry {
  let children: Entry[] = [];

  const fileNames = fs.readdirSync(dirPath);

  for (let fileName of fileNames) {
    if (ignores.test(fileName)) {
      continue;
    }

    let fileLink = dirLink + '/' + fileName;
    let filePath = path.join(dirPath, fileName);
    if (fs.statSync(filePath).isDirectory()) {
      const name = extractTitle(filePath);
      let sub = buildTree(filePath, name, fileLink);
      if (sub.children != null && sub.children.length > 0) {
        children.push(sub);
      }
    } else if (isDoc.test(fileName)) {
      const name = extractTitle(filePath);
      children.push({name: name, fileName: fileName, link: fileLink});
    }
  }

  return {name, fileName: dirPath, children, link: dirLink};
}

function renderToMd (tree: Entry, linkDir = false): string {
  if (!tree.children) {
    return `- [${tree.name || niceName(path.basename(tree.fileName, '.md'))}](${tree.link.replace(/ /g, '%20')})`;
  } else {
    let fileNames = new Set(tree.children.filter(c => !c.children).map(c => c.fileName));
    let dirNames = new Set(tree.children.filter(c => c.children).map(c => c.fileName + '.md'));

    let content = tree.children
      .filter(c => (!fileNames.has(c.fileName) || !dirNames.has(c.fileName)) && c.fileName != 'README.md')
      .map(c => renderToMd(c, dirNames.has(c.fileName + '.md') && fileNames.has(c.fileName + '.md')))
      .join('\n')
      .split('\n')
      .map(item => '  ' + item)
      .join('\n');
    let prefix = '';
    if (tree.fileName) {
      if (linkDir || fileNames.has('README.md')) {
        let linkPath = tree.link.replace(/ /g, '%20');
        if (fileNames.has('README.md')) {
          linkPath += '/README.md';
        }
        prefix = `- [${tree.name || niceName(path.basename(tree.fileName, '.md'))}](${linkPath})\n`;
      } else {
        prefix = `- ${tree.name || niceName(tree.fileName)}\n`;
      }
    }

    return prefix + content;
  }
}

function buildTOC (entry: Entry, skipFirstLevel = false) {
  if (!skipFirstLevel) {
    console.log(`Generating TOC for folder ${entry.fileName}`);
    fs.writeFileSync(path.join(entry.fileName, '_toc.md'), renderToMd(entry));
  }

  if (entry.children) {
    entry.children
      .filter((e) => fs.statSync(path.isAbsolute(e.fileName) ? e.fileName : path.join(entry.fileName, e.fileName)).isDirectory())
      .forEach((e) => buildTOC(e));
  }
}

function buildSidebar (dir: string, sidebarFileName: string = '_sidebar.md') {
  console.log(`Generating sidebar for folder ${dir}`);

  // try {
  let root = buildTree(dir, extractTitle(dir));
  fs.writeFileSync(path.join(dir, '_sidebar.md'), renderToMd(root));

  buildTOC(root);
  // } catch (e: any) {
  //   console.error('Unable to generate sidebar for directory', dir);
  //   console.error('Reason:', e.message);
  //   process.exit(1);
  // }
}

let args = yargs
  .wrap(yargs.terminalWidth() - 1)
  .usage('$0 [-d docsDir] ')
  .options({
    docsDir: {
      alias: 'd',
      type: 'string',
      describe: 'Where to look for the documentation (defaults to docs subdir of repo directory)',
    },
  }).parseSync();

let dir = path.resolve(process.cwd(), args.docsDir || './docs');

buildSidebar(dir);