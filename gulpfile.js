const gulp = require("gulp"),
      sass = require("gulp-sass"),
      mkdirp = require("mkdirp"),
      Builder = require("systemjs-builder"),
      rename = require("gulp-rename"),
      uglify = require("gulp-uglify-es").default,
      ts = require("gulp-typescript");

gulp.task("default", ["init-dist", "sass", "ts", "build", "uglify"], function() {
    console.log();
    console.log("!!! remember to remove 'node_modules' from app.yaml !!!");
    console.log();
});

gulp.task("dev-watch", function() {
    gulp.watch("./src/scss/*", ["sass"]);
    gulp.watch("./src/ts/*", ["ts"]);
});

gulp.task("init-dist", function() {
    return Promise.all([
        new Promise(r => mkdirp("./dist/js/", () => r())),
        new Promise(r => mkdirp("./dist/css/", () => r()))
    ]);
});

gulp.task("sass", ["init-dist"], function() {
    return gulp.src("./src/scss/*").
        pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError)).
        pipe(gulp.dest("./dist/css"));
});

gulp.task("ts", ["init-dist"], function() {
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
