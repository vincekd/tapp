const gulp = require("gulp");
const sass = require("gulp-sass");
const mkdirp = require("mkdirp");
const Builder = require("systemjs-builder");
const rename = require("gulp-rename");
const uglify = require("gulp-uglify-es").default;
const ts = require("gulp-typescript");

gulp.task("init-dist", initDist);
gulp.task("sass", gulp.series("init-dist", compileSass));
gulp.task("ts", gulp.series("init-dist", compileTs));
gulp.task("build", gulp.series("ts", "sass", build));
gulp.task("uglify", gulp.series("build", uglifyPkg));

gulp.task("dev-watch", gulp.series("build", function(cb) {
  process.env['NODE_ENV'] = 'development';
  gulp.watch("./src/scss/*", gulp.series("sass"));
  gulp.watch("./src/ts/**/*.ts", gulp.series("ts", build));
  cb();
}));

function initDist() {
  return Promise.all([
    new Promise(r => mkdirp("./dist/js/", () => r())),
    new Promise(r => mkdirp("./dist/css/", () => r()))
  ]);
}
function compileSass() {
  return gulp.src("./src/scss/*").
        pipe(sass({outputStyle: 'compressed'}).on('error', sass.logError)).
        pipe(gulp.dest("./dist/css"));
}
function compileTs() {
  const tsProject = ts.createProject("tsconfig.json");
  return tsProject.src().pipe(tsProject()).js.pipe(gulp.dest("./dist/js/"));
}
function build() {
  const CONFIG_FILE = "./dist/js/systemjs.config.js";
  var builder = new Builder("./");
  return new Promise(resolve => {
    builder.loadConfig(CONFIG_FILE).then(() => {
      builder.buildStatic("dist/js/main.js", "dist/js/app.js", {
        //"minify": true,
        //"format": "es6",
        "sourceMaps": process.env['NODE_ENV'] === 'development'
      }).then(() => {
        console.log("done building static app");
        resolve();
      });
    });
  });
}
function uglifyPkg() {
  return gulp.src("dist/js/app.js")
             .pipe(rename("app.min.js"))
             .pipe(uglify({
               "compress": true
             })).pipe(gulp.dest("dist/js/"));
}
