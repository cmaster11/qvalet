#!/usr/bin/env node
"use strict";
// Clearly inspired by https://github.com/hfour/docsify-tools/blob/master/src/docsify-auto-sidebar.ts
exports.__esModule = true;
var fs = require("fs");
var path = require("path");
var yargs = require("yargs");
var ignores = /node_modules|^\.|_sidebar|_docsify|_navbar|_toc.md/;
var isDoc = /.md$/;
function niceName(name) {
    var splitName = name.split('-');
    if (Number.isNaN(Number(splitName[0]))) {
        return splitName.join(' ');
    }
    return splitName.slice(1).join(' ');
}
function extractTitle(entryPath) {
    // Load file and extract the first header we find
    var readmeFile = fs.statSync(entryPath).isDirectory() ? path.join(entryPath, 'README.md') : entryPath;
    if (fs.existsSync(readmeFile)) {
        var content = fs.readFileSync(readmeFile, { encoding: 'utf-8' });
        var firstHeaderMatch = content.match(/^#+\s+(.+)$/im);
        if (firstHeaderMatch) {
            return firstHeaderMatch[1];
        }
    }
    return entryPath;
}
function buildTree(dirPath, name, dirLink) {
    if (name === void 0) { name = ''; }
    if (dirLink === void 0) { dirLink = ''; }
    var children = [];
    var fileNames = fs.readdirSync(dirPath);
    for (var _i = 0, fileNames_1 = fileNames; _i < fileNames_1.length; _i++) {
        var fileName = fileNames_1[_i];
        if (ignores.test(fileName)) {
            continue;
        }
        var fileLink = dirLink + '/' + fileName;
        var filePath = path.join(dirPath, fileName);
        if (fs.statSync(filePath).isDirectory()) {
            var name_1 = extractTitle(filePath);
            var sub = buildTree(filePath, name_1, fileLink);
            if (sub.children != null && sub.children.length > 0) {
                children.push(sub);
            }
        }
        else if (isDoc.test(fileName)) {
            var name_2 = extractTitle(filePath);
            children.push({ name: name_2, fileName: fileName, link: fileLink });
        }
    }
    return { name: name, fileName: dirPath, children: children, link: dirLink };
}
function renderToMd(tree, linkDir, skipRootLevel) {
    if (linkDir === void 0) { linkDir = false; }
    if (skipRootLevel === void 0) { skipRootLevel = false; }
    if (!tree.children) {
        return "- [" + (tree.name || niceName(path.basename(tree.fileName, '.md'))) + "](" + tree.link.replace(/ /g, '%20') + ")";
    }
    else {
        var fileNames_2 = new Set(tree.children.filter(function (c) { return !c.children; }).map(function (c) { return c.fileName; }));
        var dirNames_1 = new Set(tree.children.filter(function (c) { return c.children; }).map(function (c) { return c.fileName + '.md'; }));
        var content = tree.children
            .filter(function (c) { return (!fileNames_2.has(c.fileName) || !dirNames_1.has(c.fileName)) && c.fileName != 'README.md'; })
            .map(function (c) { return renderToMd(c, dirNames_1.has(c.fileName + '.md') && fileNames_2.has(c.fileName + '.md')); })
            .join('\n')
            .split('\n')
            .map(function (item) { return '  ' + item; })
            .join('\n');
        var prefix = '';
        if (tree.fileName && !skipRootLevel) {
            if (linkDir || fileNames_2.has('README.md')) {
                var linkPath = tree.link.replace(/ /g, '%20');
                if (fileNames_2.has('README.md')) {
                    linkPath += '/README.md';
                }
                prefix = "- [" + (tree.name || niceName(path.basename(tree.fileName, '.md'))) + "](" + linkPath + ")\n";
            }
            else {
                prefix = "- " + (tree.name || niceName(tree.fileName)) + "\n";
            }
        }
        return prefix + content;
    }
}
function buildTOC(entry, skipFirstLevel) {
    if (skipFirstLevel === void 0) { skipFirstLevel = false; }
    if (!skipFirstLevel) {
        console.log("Generating TOC for folder " + entry.fileName);
        fs.writeFileSync(path.join(entry.fileName, '_toc.md'), renderToMd(entry, false, true));
    }
    if (entry.children) {
        entry.children
            .filter(function (e) { return fs.statSync(path.isAbsolute(e.fileName) ? e.fileName : path.join(entry.fileName, e.fileName)).isDirectory(); })
            .forEach(function (e) { return buildTOC(e); });
    }
}
function buildSidebar(dir, sidebarFileName) {
    if (sidebarFileName === void 0) { sidebarFileName = '_sidebar.md'; }
    console.log("Generating sidebar for folder " + dir);
    // try {
    var root = buildTree(dir, extractTitle(dir));
    fs.writeFileSync(path.join(dir, '_sidebar.md'), renderToMd(root));
    buildTOC(root);
    // } catch (e: any) {
    //   console.error('Unable to generate sidebar for directory', dir);
    //   console.error('Reason:', e.message);
    //   process.exit(1);
    // }
}
var args = yargs
    .wrap(yargs.terminalWidth() - 1)
    .usage('$0 [-d docsDir] ')
    .options({
    docsDir: {
        alias: 'd',
        type: 'string',
        describe: 'Where to look for the documentation (defaults to docs subdir of repo directory)'
    }
}).parseSync();
var dir = path.resolve(process.cwd(), args.docsDir || './docs');
buildSidebar(dir);
