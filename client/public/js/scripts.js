/*

    countUp.js
    by @inorganik

*/

// target = id of html element or var of previously selected html element where counting occurs
// startVal = the value you want to begin at
// endVal = the value you want to arrive at
// decimals = number of decimal places, default 0
// duration = duration of animation in seconds, default 2
// options = optional object of options (see below)

var CountUp = function(target, startVal, endVal, decimals, duration, options) {

    // make sure requestAnimationFrame and cancelAnimationFrame are defined
    // polyfill for browsers without native support
    // by Opera engineer Erik Möller
    var lastTime = 0;
    var vendors = ['webkit', 'moz', 'ms', 'o'];
    for(var x = 0; x < vendors.length && !window.requestAnimationFrame; ++x) {
        window.requestAnimationFrame = window[vendors[x]+'RequestAnimationFrame'];
        window.cancelAnimationFrame =
          window[vendors[x]+'CancelAnimationFrame'] || window[vendors[x]+'CancelRequestAnimationFrame'];
    }
    if (!window.requestAnimationFrame) {
        window.requestAnimationFrame = function(callback, element) {
            var currTime = new Date().getTime();
            var timeToCall = Math.max(0, 16 - (currTime - lastTime));
            var id = window.setTimeout(function() { callback(currTime + timeToCall); },
              timeToCall);
            lastTime = currTime + timeToCall;
            return id;
        };
    }
    if (!window.cancelAnimationFrame) {
        window.cancelAnimationFrame = function(id) {
            clearTimeout(id);
        };
    }

     // default options
    this.options = {
        useEasing : true, // toggle easing
        useGrouping : true, // 1,000,000 vs 1000000
        separator : ',', // character to use as a separator
        decimal : '.' // character to use as a decimal
    };
    // extend default options with passed options object
    for (var key in options) {
        if (options.hasOwnProperty(key)) {
            this.options[key] = options[key];
        }
    }
    if (this.options.separator === '') this.options.useGrouping = false;
    if (!this.options.prefix) this.options.prefix = '';
    if (!this.options.suffix) this.options.suffix = '';

    this.d = (typeof target === 'string') ? document.getElementById(target) : target;
    this.startVal = Number(startVal);
    this.endVal = Number(endVal);
    this.countDown = (this.startVal > this.endVal);
    this.frameVal = this.startVal;
    this.decimals = Math.max(0, decimals || 0);
    this.dec = Math.pow(10, this.decimals);
    this.duration = Number(duration) * 1000 || 2000;
    var self = this;

    this.version = function () { return '1.6.0'; };

    // Print value to target
    this.printValue = function(value) {
        var result = (!isNaN(value)) ? self.formatNumber(value) : '--';
        if (self.d.tagName == 'INPUT') {
            this.d.value = result;
        }
        else if (self.d.tagName == 'text') {
            this.d.textContent = result;
        }
        else {
            this.d.innerHTML = result;
        }
    };

    // Robert Penner's easeOutExpo
    this.easeOutExpo = function(t, b, c, d) {
        return c * (-Math.pow(2, -10 * t / d) + 1) * 1024 / 1023 + b;
    };
    this.count = function(timestamp) {

        if (!self.startTime) self.startTime = timestamp;

        self.timestamp = timestamp;

        var progress = timestamp - self.startTime;
        self.remaining = self.duration - progress;

        // to ease or not to ease
        if (self.options.useEasing) {
            if (self.countDown) {
                self.frameVal = self.startVal - self.easeOutExpo(progress, 0, self.startVal - self.endVal, self.duration);
            } else {
                self.frameVal = self.easeOutExpo(progress, self.startVal, self.endVal - self.startVal, self.duration);
            }
        } else {
            if (self.countDown) {
                self.frameVal = self.startVal - ((self.startVal - self.endVal) * (progress / self.duration));
            } else {
                self.frameVal = self.startVal + (self.endVal - self.startVal) * (progress / self.duration);
            }
        }

        // don't go past endVal since progress can exceed duration in the last frame
        if (self.countDown) {
            self.frameVal = (self.frameVal < self.endVal) ? self.endVal : self.frameVal;
        } else {
            self.frameVal = (self.frameVal > self.endVal) ? self.endVal : self.frameVal;
        }

        // decimal
        self.frameVal = Math.round(self.frameVal*self.dec)/self.dec;

        // format and print value
        self.printValue(self.frameVal);

        // whether to continue
        if (progress < self.duration) {
            self.rAF = requestAnimationFrame(self.count);
        } else {
            if (self.callback) self.callback();
        }
    };
    // start your animation
    this.start = function(callback) {
        self.callback = callback;
        self.rAF = requestAnimationFrame(self.count);
        return false;
    };
    // toggles pause/resume animation
    this.pauseResume = function() {
        if (!self.paused) {
            self.paused = true;
            cancelAnimationFrame(self.rAF);
        } else {
            self.paused = false;
            delete self.startTime;
            self.duration = self.remaining;
            self.startVal = self.frameVal;
            requestAnimationFrame(self.count);
        }
    };
    // reset to startVal so animation can be run again
    this.reset = function() {
        self.paused = false;
        delete self.startTime;
        self.startVal = startVal;
        cancelAnimationFrame(self.rAF);
        self.printValue(self.startVal);
    };
    // pass a new endVal and start animation
    this.update = function (newEndVal) {
        cancelAnimationFrame(self.rAF);
        self.paused = false;
        delete self.startTime;
        self.startVal = self.frameVal;
        self.endVal = Number(newEndVal);
        self.countDown = (self.startVal > self.endVal);
        self.rAF = requestAnimationFrame(self.count);
    };
    this.formatNumber = function(nStr) {
        nStr = nStr.toFixed(self.decimals);
        nStr += '';
        var x, x1, x2, rgx;
        x = nStr.split('.');
        x1 = x[0];
        x2 = x.length > 1 ? self.options.decimal + x[1] : '';
        rgx = /(\d+)(\d{3})/;
        if (self.options.useGrouping) {
            while (rgx.test(x1)) {
                x1 = x1.replace(rgx, '$1' + self.options.separator + '$2');
            }
        }
        return self.options.prefix + x1 + x2 + self.options.suffix;
    };

    // format startVal on initialization
    self.printValue(self.startVal);
};

// Example:
// var numAnim = new countUp("SomeElementYouWantToAnimate", 0, 99.99, 2, 2.5);
// numAnim.start();
// numAnim.update(135);
// with optional callback:
// numAnim.start(someMethodToCallOnComplete);

/*!
 *
 * DBOT - Demisto Security Bot
 *
 * Author: @demisto
 * Website: https://www.demisto.com
 * License: https://raw.github.com/demisto/alfred-web/master/LICENSE
 *
 */

(function ($) {
  'use strict';

  if (typeof $ === 'undefined') { throw new Error('This site\'s JavaScript requires jQuery'); }

  // cache common elements
  var $win  = $(window);
  var $doc  = $(document);
  var $body = $('body');



  $win.on('load', function() {
    $body.addClass('site-loaded loaded');
  });


  // Show sticky topbar on scroll
  // -----------------------------------

  var stickyNavScroll;
  var stickySelector = '#header .navbar-sticky';

  // Setup functions based on screen
  if (matchMedia('(min-width: 992px), (max-width: 767px)').matches) {
    stickyNavScroll = function () {
      var top = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
      var features_section_top = $("#features").offset().top;
      var features_section_bottom = features_section_top + $("#features").outerHeight();


      if (top > 40) {
        if (!$(stickySelector).hasClass('navbar-sticky-color')) {
          // change the transparency
          $(stickySelector).stop().css('top', '-80px');
          $(stickySelector).removeClass('navbar-sticky-color-trans');
          $(stickySelector).addClass('navbar-sticky-color');
          $(stickySelector).stop().animate({'top': '0'});
        }
      }
      else {
        $(stickySelector).removeClass('navbar-sticky-color');
        $(stickySelector).addClass('navbar-sticky-color-trans')
        $(stickySelector).stop().css('top', '0px');
      }
    };
  }

  if (matchMedia('(min-width: 768px) and (max-width: 991px)').matches) {
    stickyNavScroll = function () {
      var top = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;

      if (social_widget_bottom > features_section_top && social_widget_top < features_section_bottom) {
        $('#social').addClass('socialwidget-light');
        $('#social').removeClass('socialwidget-dark');
      } else {
        $('#social').addClass('socialwidget-dark');
        $('#social').removeClass('socialwidget-light');
      }


      if (top > 40) {
        if (!$(stickySelector).hasClass('navbar-sticky-color')) {
          // change the transparency
          $(stickySelector).stop().css('top', '-120px');
          $(stickySelector).addClass('navbar-sticky-color');
          $(stickySelector).removeClass('navbar-sticky-color-trans');
          $(stickySelector).stop().animate({'top': '0'});
        }
      }
      else {
        $(stickySelector).removeClass('navbar-sticky-color');
        $(stickySelector).addClass('navbar-sticky-color-trans')
        $(stickySelector).stop().css('top', '0px');
      }
    };
  }

  // Finally attach to events
  // attach only on home page
  if ($('#slack-message').length) {
    $doc.ready(stickyNavScroll);
    $win.scroll(stickyNavScroll);
  }

  // Sticky Navigation
  // -----------------------------------

  $(function() {

    $('.main-navbar').onePageNav({
      scrollThreshold: 0.25,
      filter: ':not(.external)', // external links
      changeHash: true,
      scrollSpeed: 750
    });

  });


  // Video Background
  // -----------------------------------
  //
  $(function() {
    if (matchMedia('(min-width: 1024px)').matches) {
      if ($('#video').length) {
        var videobackground = new $.backgroundVideo( $('#video'), {
          'align':    'centerXY',
          'width':    1280,
          'height':   720,
          'path':     'img/',
          'filename': 'videobg',
          'types':    ['mp4']
        });
      }
    }
  });

// appshot owl-carousel
  $('#appshots').owlCarousel({

      autoPlay: 3000,
      items: 2,
      itemsDesktop : [1199,3],
      itemsDesktopSmall : [979,3],
      margin:          10,
      responsiveClass: true,
      responsive:      {
          0: {
              items: 1,
              nav:   false
          },
          500: {
              items: 2,
              nav:   false
          },
          1000: {
              items: 4,
              nav:   false,
              loop:  false
          }
      }
  });


  // Self close navbar on mobile click
  // -----------------------------------
  $(function(){
       var navMain = $("#navbar-main");
       var navToggle = $('.navbar-toggle');

       navMain.on('click', 'a', null, function () {
         if (!($(this).attr('id') == 'namelink')) {
           if ( navToggle.is(':visible') )
             navMain.collapse('hide');
         }
       });
   });


  // Check if we already have the cookie and if so, change the title of the button
  // -----------------------------------
  $(function () {
    // If we are on the homepage
    if ($('#counter').length) {
      var counter_timer;
      var last_stop_counter;
      var refreshCounter = function(start_count, stop_count, duration) {
        var options = {
          useEasing : false,
          useGrouping : true,
          separator : ' ',
          decimal : '.',
          prefix : '',
          suffix : ''
        };
        var counter = new CountUp("counter", start_count, stop_count, 0, duration, options);
        last_stop_counter = stop_count;
        counter.start();
      };

      $.getJSON('/messages', function(data) {
        var s_count = Math.max(data.count - 100, 0);
        var st_count = data.count;
        refreshCounter(s_count, st_count, 60);
        counter_timer = setInterval(function() {
          $.getJSON('/messages', function(data) {
            var s_count = last_stop_counter;
            var st_count = data.count;
            refreshCounter(s_count, st_count, 60);
          });
        }, 60000);
      });

      $(window).on('beforeunload', function() {
        if (counter_timer) {
          clearInterval(counter_timer);
        }
      });



      $.getJSON('/user', function(data) {
        $('#slack-message').html('Configure D<small>BOT</small>');
        $('#slack-message').attr('href', '/conf');
        $('#slack-message').attr('target', '');
      });
      var recaptcha_widget_id = null;
      var captcha_callback = function(captcha_response) {
        var emailaddress = $('#emailaddress').val();
        var save = {'email':emailaddress , 'captcharesponse':captcha_response}
        $.ajax({
          type: 'POST',
          url: '/join',
          data: JSON.stringify(save),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(){
            $('#recaptchaLabel').html('Slack Channel Subscribed.');
            $('#recaptchadiv').html('Your subscription is successful. We will send you an invite soon, see you on Slack channel.');
            $('#emailaddress').val('');
            window.setTimeout(function() {
              $('#recaptcha').modal('hide');
            }, 5000);
          },
          error: function(xhr, status, error) {
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              if (xhr.responseJSON.errors[0].status === 400 && xhr.responseJSON.errors[0].id === 'bad_captcha') {
                if (recaptcha_widget_id != null) {
                  grecaptcha.reset(recaptcha_widget_id);
                  $('#recaptchadiv').append('Error while validating. Please try again.');
                  return;
                }
              }
              err += " - " + xhr.responseJSON.errors[0].detail;
            }
            $('#recaptchaLabel').html('Error While Subscribing');
            $('#recaptchadiv').html(err + ' - this error has been logged and we are on it.');
          }
        });
      }

      // recaptcha
      var render_captcha = function() {
        // Clean the recaptcha
        $('#recaptchadiv').html('');
        recaptcha_widget_id = grecaptcha.render('recaptchadiv', { 'sitekey' : '6Let7QsTAAAAAG90E160XQZtIGOWyh59nTefLXFx', 'callback': captcha_callback});
      }

      $('#join').submit(function(event) {
        event.preventDefault();
        if ($('#emailaddress')[0].checkValidity()) {
          $('#recaptchaLabel').html('Please confirm you are a human');
          $('#recaptchadiv').html();
          $('#recaptcha').modal('show');
        }
      });

      $('#recaptcha').on('shown.bs.modal', function () {
          render_captcha();
      });
    }
  });

  window.logout = function() {
    $.getJSON('/logout', function() {
      var loc = window.location;
      loc.href = loc.protocol + '//' + loc.host;
    });
  };
})(window.jQuery);


// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';

  if ($('#details').length) {
    $(window).on("scroll", function() {
      if ($(this).scrollTop() < $('.site-header').height()) {
        $('.site-header').removeClass("fixed-header-menu");
      } else if ($(this).scrollTop() > $('.site-header').height()) {
        $('.site-header').addClass("fixed-header-menu");
      };

      if ($(this).scrollTop() < 750) {
        $('#scrollUp').removeClass("show");
      } else if ($(this).scrollTop() > 750) {
        $('#scrollUp').addClass("show");
      }
    });
  }
})(window.jQuery);
