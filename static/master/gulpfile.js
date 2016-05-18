var gulp        = require('gulp'),
    concat      = require('gulp-concat'),
    uglify      = require('gulp-uglify'),
    jade        = require('gulp-jade'),
    less        = require('gulp-less'),
    path        = require('path'),
    livereload  = require('gulp-livereload'), // Livereload plugin needed: https://chrome.google.com/webstore/detail/livereload/jnihajbhpnppcggbcgedagnkighmdlei
    marked      = require('marked'), // For :markdown filter in jade
    path        = require('path'),
    changed     = require('gulp-changed'),
    prettify    = require('gulp-html-prettify'),
    rename      = require('gulp-rename'),
    through     = require('through2'),
    gutil       = require('gulp-util'),
    minifyCSS   = require('gulp-minify-css'),
    gulpFilter  = require('gulp-filter'),
    expect      = require('gulp-expect-file'),
    gulpsync    = require('gulp-sync')(gulp),
    sourcemaps  = require('gulp-sourcemaps'),
    react       = require('gulp-react'),
    PluginError = gutil.PluginError;
    debug       = require('gulp-debug');
    replace     = require('gulp-replace');
    gtmidsrc    = require('./\_gtmid.json');

// production mode (see build task)
var isProduction = false;
var useSourceMaps = false;

// ignore everything that begins with underscore
var hidden_files = '**/_*.*';
var ignored_files = '!'+hidden_files;

// VENDOR CONFIG
var vendor = {
  site: {
    source: require('./vendor.json'),
    dest: '../site/vendor'
  }
};

// SOURCES CONFIG
var source = {
  scripts: {
    site:  [ 'js/countUp.js',
             'js/script.js',
             'js/main.js',
             'js/modules/**/*.js',
             'js/custom/**/*.js',
              ignored_files
            ],
    newfiles: ['js/vendor/**/*'],
    watch: ['js/*.js', 'js/**/*.js']
  },
  templates: {
    pages: {
        files : ['jade/*.jade', ignored_files, 'index.html', 'slackuser.html'],
        watch: ['jade/**/*.jade', 'jade/*.jade', 'index.html', 'slackuser.html', 'jade/'+hidden_files]
    }
  },
  styles: {
    site: {
      main: ['less/styles.less','less/main.less', '!less/themes/*.less'],
      dir:  'less',
      watch: ['less/*.less', 'less/**/*.less', '!less/themes/*.less']
    },
    themes: {
      main: ['less/themes/*.less', ignored_files],
      dir:  'less/themes',
      watch: ['less/themes/*.less']
    },
  },
  bootstrap: {
    main: 'less/bootstrap/bootstrap.less',
    dir:  'less/bootstrap',
    watch: ['less/bootstrap/*.less']
  }
};

// BUILD TARGET CONFIG
var build = {
  scripts: {
    site: {
      main: 'scripts.js',
      dir: '../site/js'
    },
    vendor: {
      main: 'vendor.js',
      dir: '../vendor/js'
    }
  },
  styles: '../site/css/',
  templates: {
    pages: '../site'
  }
};




//---------------
// TASKS
//---------------



// JS SITE
gulp.task('scripts:site', function() {
    // Minify and copy all JavaScript (except vendor scripts)
    return gulp.src(source.scripts.site)
        .pipe(react())
        .pipe( useSourceMaps ? sourcemaps.init() : gutil.noop())
        .pipe(concat(build.scripts.site.main))
        .on("error", handleError)
        .pipe( isProduction ? uglify({preserveComments:'some'}) : gutil.noop() )
        .on("error", handleError)
        .pipe( useSourceMaps ? sourcemaps.write() : gutil.noop() )
        .pipe(gulp.dest(build.scripts.site.dir));
});


// VENDOR BUILD

// copy file from bower folder into the site vendor folder
gulp.task('scripts:vendor', function() {

  var jsFilter = gulpFilter('**/*.js');

  return gulp.src(vendor.site.source, {base: 'bower_components'})
      .pipe(expect(vendor.site.source))
      .pipe(jsFilter)
      .pipe(concat(build.scripts.vendor.main))
      .on("error", handleError)
      .pipe( isProduction ? uglify({preserveComments:'some'}) : gutil.noop() )
      .on("error", handleError)
      .pipe(gulp.dest(build.scripts.site.dir));

});

// copy file from js/vendor as is - new is new build stuff - needs cleanup later. 
gulp.task('scripts:newvendor', function() {
  return gulp.src(source.scripts.newfiles)
      .pipe(debug())
      .pipe(gulp.dest(build.scripts.site.dir));
});


// copy file from bower folder into the site vendor folder
gulp.task('styles:vendor', function() {
  var cssFilter = gulpFilter('**/*.css');

  return gulp.src(vendor.site.source.concat(["!bower_components/**/*.js"]), {base: 'bower_components'})
      .pipe(cssFilter)
      .pipe( isProduction ? minifyCSS() : gutil.noop() )
      .pipe(cssFilter.restore())
      .pipe( gulp.dest(vendor.site.dest) );

});


// SITE LESS
gulp.task('styles:site', function() {
    return gulp.src(source.styles.site.main)
        .pipe( useSourceMaps ? sourcemaps.init() : gutil.noop())
        .pipe(less({
            paths: [source.styles.site.dir]
        }))
        .on("error", handleError)
        .pipe( isProduction ? minifyCSS() : gutil.noop() )
        .pipe( useSourceMaps ? sourcemaps.write() : gutil.noop())
        .pipe(gulp.dest(build.styles));
});

// LESS THEMES
gulp.task('styles:themes', function() {
    return gulp.src(source.styles.themes.main)
        .pipe(less({
            paths: [source.styles.themes.dir]
        }))
        .on("error", handleError)
        .pipe(gulp.dest(build.styles));
});

// BOOSTRAP
gulp.task('bootstrap', function() {
    return gulp.src(source.bootstrap.main)
        .pipe(less({
            paths: [source.bootstrap.dir]
        }))
        .on("error", handleError)
        .pipe(gulp.dest(build.styles));
});


// JADE
gulp.task('templates:pages', function() {
    var jadefilter = gulpFilter(['**/*.jade'], {restore: true});

    return gulp.src(source.templates.pages.files)
        .pipe(jadefilter)
        .pipe(jade())
        .pipe(jadefilter.restore())
        .pipe(changed(build.templates.pages, { extension: '.html' }))
        .pipe(debug({extension: '.html'}))
        .pipe(replace('gtmid', gtmidsrc.gtmid))
        .on("error", handleError)
        .pipe(prettify({
            indent_char: ' ',
            indent_size: 3,
            unformatted: ['a', 'sub', 'sup', 'b', 'i', 'u', 'small']
        }))
        .pipe(gulp.dest(build.templates.pages))
        ;
});


//---------------
// WATCH
//---------------

// Rerun the task when a file changes
gulp.task('watch', function() {
  livereload.listen();

  gulp.watch(source.scripts.watch,           ['scripts:site']);
  gulp.watch(source.styles.site.watch,       ['styles:site']);
  gulp.watch(source.styles.themes.watch,     ['styles:themes']);
  gulp.watch(source.bootstrap.watch,         ['styles:site']); //bootstrap
  gulp.watch(source.templates.pages.watch,   ['templates:pages']);

  gulp.watch([

      '../site/**'

  ]).on('change', function(event) {

      livereload.changed( event.path );

  });

});


//---------------
// DEFAULT TASK
//---------------

// build for production (minify)
gulp.task('build', ['prod', 'default-finish']);
gulp.task('prod', function() { isProduction = true; });
// build with sourcemaps (no minify)
gulp.task('sourcemaps', ['usesources', 'default']);
gulp.task('usesources', function(){ useSourceMaps = true; });
// default (no minify)
gulp.task('start',[
          'styles:site',
          'styles:themes',
          'styles:vendor',
          'templates:pages',
          'watch'
        ]);

gulp.task('finish',[
          'styles:site',
          'styles:themes',
          'styles:vendor',
          'templates:pages'
        ]);

gulp.task('default', gulpsync.sync([
          'scripts:vendor',
          'scripts:site',
          'scripts:newvendor',
          'start'
        ]), function(){

  gutil.log(gutil.colors.cyan('************'));
  gutil.log(gutil.colors.cyan('* All Done *'), 'You can start editing your code, LiveReload will update your browser after any change..');
  gutil.log(gutil.colors.cyan('************'));

});

gulp.task('default-finish', gulpsync.sync([
          'scripts:vendor',
          'scripts:site',
          'finish'
        ]), function(){

  gutil.log(gutil.colors.cyan('************'));
  gutil.log(gutil.colors.cyan('* All Done *'));
  gutil.log(gutil.colors.cyan('************'));

});


gulp.task('done', function(){
  console.log('All Done!! You can start editing your code, LiveReload will update your browser after any change..');
});

// Error handler
function handleError(err) {
  console.log(err.toString());
  this.emit('end');
}
