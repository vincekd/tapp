const gulp = require("gulp"),
      sass = require("node-sass"),
      fs = require("fs"),
      Builder = require("systemjs-builder"),
      rename = require("gulp-rename"),
      uglify = require("gulp-uglify-es").default,
      ts = require("gulp-typescript");

gulp.task("default", ["sass", "ts", "build", "uglify"], function() {
    //TODO: create dist/js dist/css if not exist
    console.log();
    console.log("!!! remember to remove 'node_modules' from app.yaml !!!");
    console.log();
});

gulp.task("sass", function() {
    const scss = ["main", "theme", "admin"];
    // compile scss
    scss.forEach(file => {
        sass.render({
            "file": `src/scss/${file}.scss`,
            "outFile": `dist/css/${file}.css`,
            "outputStyle": "compressed"
        }, (err, result) => {
            if (err) {
                console.error(err);
            } else {
                fs.writeFile(`dist/css/${file}.css`, result.css, err => {
                    if (err) {
                        console.error(err);
                    }
                });
            }
        });
    });
});

gulp.task("ts", function() {
    let tsProject = ts.createProject("tsconfig.json");
    return tsProject.src().pipe(tsProject()).js.pipe(gulp.dest("./dist/js/"));
});

gulp.task("build", ["ts"], function() {
    const CONFIG_FILE = "./dist/js/systemjs.config.js";
    var builder = new Builder("./");
    return new Promise(resolve => {
        builder.loadConfig(CONFIG_FILE).then(() => {
            builder.buildStatic("dist/js/main.js", "dist/js/app.js", {
                //"minify": true,
                //"format": "es6",
                "sourceMaps": false
            }).then(() => {
                console.log("done building static app");
                resolve();
            });
        });
    });
});

gulp.task("uglify", ["build"], function() {
    return gulp.src("dist/js/app.js")
        .pipe(rename("app.min.js"))
        .pipe(uglify({
            "compress": true
        })).pipe(gulp.dest("dist/js/"));
});
