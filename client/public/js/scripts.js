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
    $body.addClass('site-loaded');
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

  // Run this only on details page
  if ($('#details').length) {
    var MD5Mask = 1;
    var URLMask = 2;
    var IPMask = 4;
    var FILEMask = 8;

    var isfile;
    var ismd5;
    var isurl;
    var isip;

    var uri = new URI();
    var qParts = uri.search(true);

    // freshdesk widget
    $.ajax({
      type: 'GET',
      url: '/user',
      data: "",
      dataType: 'json',
      contentType: 'application/json; charset=utf-8',
      success: function(data) {

        $('#realname').text(data.real_name);
        $('#useremail').text(data.email);
        $('#teamname').text('Team: ' + data.team_name);

        zE(function() {
          zE.identify({
            name: data.real_name,
            email: data.email
          });
        });

      },
      error: function(xhr, status, error) {

      }
    });

    var sortByFirstSeen = function(a, b) {
      return a.firstseen > b.firstseen ? -1 : a.firstseen < b.firstseen ? 1 : 0;
    }

    var sortByCreate = function(a, b) {
      return a.created > b.created ? -1 : a.created < b.created ? 1 : 0;
    }

    var sortByScanDate = function(a, b) {
      return a.scan_date > b.scan_date ? -1 : a.scan_date < b.scan_date ? 1 : 0;
    }

    var sortByLastResolved = function(a, b) {
      return a.last_resolved > b.last_resolved ? -1 : a.last_resolved < b.last_resolved ? 1 : 0;
    }

    var arrOrUnknown = function(arr) {
      if (arr) {
        return arr;
      }
      return ['Unknown'];
    }

    var mapOrUnknown = function(m) {
      if (m) {
        return m;
      }
      return {'Unknown': true};
    }

    // ======================== IP section ===========================

    var IPDiv = React.createClass({displayName: "IPDiv",
      render: function() {
        if (!isip) {
          return null;
        }
        else {
          var ipdata = this.props.data.ips[0];
          return (
            React.createElement("div", null,
              React.createElement("h2", null, "IP: ", ipdata.details),
              React.createElement(IPResultMessage, {data: ipdata}),
              React.createElement(IPDetails, {data: ipdata})
            )
          );
        }
      }
    });

    var IPResultMessage = React.createClass({displayName: "IPResultMessage",
      render: function() {
        var ipdata = this.props.data;
        var resultMessage = 'Could not determine the IP address reputation.';
        var color = 'warning-text';

        if (ipdata.private) {
          resultMessage = 'IP address is a private (internal) IP - no reputation found.';
          color = 'success-text';
        } else if (ipdata.Result == 0) {
          resultMessage = 'IP address is found to be clean.';
          color = 'success-text';
        } else if (ipdata.Result == 1) {
          resultMessage = "IP address is found to be malicious.";
          color = 'danger-text';
        }
        return (React.createElement("h3", {className: color}, resultMessage))
      }
    });

    var IPDetails = React.createClass({displayName: "IPDetails",
      render: function() {
        var data = this.props.data;
        if (data &&
          (data.xfe && !data.xfe.not_found ||
          data.vt.ip_report && data.vt.ip_report.response_code === 1)) {
          return (
            React.createElement("div", {className: "panel-group", id: "ipproviders", role: "tablist", "aria-multiselectable": "true"},
              React.createElement(IPXFE, {data: data}),
              React.createElement(IPVT, {data: data})
            )
          );
        }
        return null;
      }
    });

    var IPXFE = React.createClass({displayName: "IPXFE",
      render: function() {
        var data = this.props.data;
        if (data && data.xfe && !data.xfe.not_found) {
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hipxfe"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {role: "button", "data-toggle": "collapse", "data-parent": "#ipproviders", href: "#ipxfe", "aria-expanded": "true", "aria-controls": "ipxfe"},
                    "IBM X-Force Exchange Data"
                  )
                )
              ),
              React.createElement("div", {id: "ipxfe", className: "panel-collapse collapse in", role: "tabpanel", "aria-labelledby": "hipxfe"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement("h3", null, " Risk Score: ", data.xfe.ip_reputation.score),
                  React.createElement("h3", null, " Country: ", data.xfe.ip_reputation.geo && data.xfe.ip_reputation.geo['country'] ? data.xfe.ip_reputation.geo['country'] : 'Unknown', " "),
                  React.createElement("h3", null, " Categories: ", Object.keys(mapOrUnknown(data.xfe.ip_reputation.cats)).join(', '), " "),
                  React.createElement(SubnetSection, {data: data.xfe.ip_reputation.subnets}),
                  React.createElement(IPHistory, {data: data.xfe.ip_history.history})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var SubnetSection = React.createClass({displayName: "SubnetSection",
      render: function() {
        var rows = [];
        var subnets = this.props.data;
        if (subnets && subnets.length > 0) {
          subnets.sort(sortByCreate);
          for (var i=0; i < subnets.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'ipr_subnet_' + i},
                React.createElement("td", null, subnets[i].subnet),
                React.createElement("td", null, subnets[i].score),
                React.createElement("td", null, Object.keys(mapOrUnknown(subnets[i].cats)).join(', ')),
                React.createElement("td", null, subnets[i].geo && subnets[i].geo['country'] ? subnets[i].geo['country'] : 'Unknown'),
                React.createElement("td", null, subnets[i].reason),
                React.createElement("td", null, subnets[i].created)
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Subnets"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null,
                  React.createElement("th", null, "Subnet"),
                  React.createElement("th", null, "Score"),
                  React.createElement("th", null, "Category"),
                  React.createElement("th", null, "Location"),
                  React.createElement("th", null, "Reason"),
                  React.createElement("th", null, "Created")
                ),
                React.createElement("tbody", null,
                  rows
                )
              )
            )
          );
        }
        return null;
      }
    });

    var IPHistory = React.createClass({displayName: "IPHistory",
      render: function() {
        var rows = [];
        var history = this.props.data;
        if (history && history.length > 0) {
          history.sort(sortByCreate);
          for (var i=0; i < history.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'ipr_hist_' + i},
                React.createElement("td", null, history[i].ip),
                React.createElement("td", null, history[i].score),
                React.createElement("td", null, Object.keys(mapOrUnknown(history[i].cats)).join(', ')),
                React.createElement("td", null, history[i].geo && history[i].geo['country'] ? history[i].geo['country'] : 'Unknown'),
                React.createElement("td", null, history[i].reason),
                React.createElement("td", null, history[i].created)
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "IP History"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null,
                  React.createElement("th", null, "IP"),
                  React.createElement("th", null, "Score"),
                  React.createElement("th", null, "Category"),
                  React.createElement("th", null, "Location"),
                  React.createElement("th", null, "Reason"),
                  React.createElement("th", null, "Created")
                ),
                React.createElement("tbody", null,
                  rows
                )
              )
            )
          );
        }
        return null;
      }
    });

    var IPVT = React.createClass({displayName: "IPVT",
      render: function() {
        var data = this.props.data;
        if (data && data.vt && data.vt.ip_report && data.vt.ip_report.response_code === 1) {
          var xfeFound = data.xfe && !data.xfe.not_found;
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hurlvt"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {className: xfeFound ? 'collapsed' : '', role: "button", "data-toggle": "collapse", "data-parent": "#ipproviders", href: "#ipvt", "aria-expanded": "true", "aria-controls": "ipvt"},
                    "Virus Total Data"
                  )
                )
              ),
              React.createElement("div", {id: "ipvt", className: xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in', role: "tabpanel", "aria-labelledby": "hipvt"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement(ResolutionSection, {data: data.vt.ip_report.Resolutions}),
                  React.createElement(DetectedURLSection, {data: data.vt.ip_report.detected_urls})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var DetectedURLSection = React.createClass({displayName: "DetectedURLSection",
      render: function() {
        var detected = this.props.data;
        if (detected && detected.length > 0) {
          detected.sort(sortByScanDate);
          var rows = [];
          for (var i=0; i < detected.length && i < 10; i++) {
            rows.push(React.createElement("tr", {key: 'ip_detected_' + i}, React.createElement("td", null, detected[i].url), React.createElement("td", null, detected[i].positives, " / ", detected[i].total), React.createElement("td", null, detected[i].scan_date)));
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Detected URLs"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null,
                  React.createElement("th", {style: {width:'70%'}}, "URL"),
                  React.createElement("th", null, "Positives"),
                  React.createElement("th", null, "Scan Date")
                ),
                React.createElement("tbody", null,
                  rows
                )
              )
            )
          );
        }
        return null;
      }
    });

    var ResolutionSection = React.createClass({displayName: "ResolutionSection",
      render: function() {
        var resArr = this.props.data;
        if (resArr && resArr.length > 0) {
          resArr.sort(sortByLastResolved);
          var rows = [];
          for (var i=0; i<resArr.length && i < 10; i++) {
            rows.push(React.createElement("tr", {key: 'resolv_' + i}, React.createElement("td", null, resArr[i].hostname), React.createElement("td", null, resArr[i].last_resolved)));
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Historical Resolutions"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null,
                  React.createElement("th", null, "Hostname"),
                  React.createElement("th", null, "Last Resolved")
                ),
                React.createElement("tbody", null,
                  rows
                )
              )
            )
          );
        }
        return null;
      }
    });
    // ======================== END IP section ===========================

    // ======================== URL section ===========================
    var URLDiv = React.createClass({displayName: "URLDiv",
      render: function() {
        if (!isurl) {
          return null;
        } else {
          var urldata = this.props.data.urls[0];
          return (
            React.createElement("div", null,
              React.createElement("h2", null, "URL: ", urldata.details),
              React.createElement(URLResultMessage, {data: urldata}),
              React.createElement(URLDetails, {urldata: urldata})
            )
          );
        }
      }
    });

    var URLResultMessage = React.createClass({displayName: "URLResultMessage",
      render: function() {
        var urldata = this.props.data;
        var resultMessage = 'Could not determine the URL reputation.';
        var color = 'warning-text';

        if (urldata.Result == 0)
        {
          resultMessage = 'URL address is found to be clean.';
          color = 'success-text';
        }
        else if (urldata.Result == 1)
        {
          resultMessage = 'URL address is found to be malicious.';
          color = 'danger-text';
        }
        return (React.createElement("h3", {className: color}, resultMessage))
      }
    });

    var URLDetails = React.createClass({displayName: "URLDetails",
      render: function() {
        var urldata = this.props.urldata;
        if (urldata &&
          (urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A) ||
          urldata.vt.url_report && urldata.vt.url_report.response_code === 1)) {
          return (
            React.createElement("div", {className: "panel-group", id: "urlproviders", role: "tablist", "aria-multiselectable": "true"},
              React.createElement(URLXFE, {urldata: urldata}),
              React.createElement(URLVT, {urldata: urldata})
            )
          );
        }
        return null;
      }
    });

    var URLXFE = React.createClass({displayName: "URLXFE",
      render: function() {
        var urldata = this.props.urldata;
        if (urldata && urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A)) {
          var mxToStr = function(mx) {
            return mx.exchange + '(' + mx.priority + ')';
          }
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hurlxfe"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {role: "button", "data-toggle": "collapse", "data-parent": "#urlproviders", href: "#urlxfe", "aria-expanded": "true", "aria-controls": "urlxfe"},
                    "IBM X-Force Exchange Data"
                  )
                )
              ),
              React.createElement("div", {id: "urlxfe", className: "panel-collapse collapse in", role: "tabpanel", "aria-labelledby": "hurlxfe"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement(URLRiskScore, {data: urldata.xfe}),
                  React.createElement(URLCategory, {urldata: urldata}),
                  React.createElement("table", {className: "table"},
                    React.createElement("thead", null, React.createElement("th", null, "Name"), React.createElement("th", null, "Value")),
                    React.createElement("tbody", null,
                      React.createElement(TDRecord, {t: "A Records", arr: urldata.xfe.resolve.A}),
                      React.createElement(TDRecord, {t: "AAAA Records", arr: urldata.xfe.resolve.AAAA}),
                      React.createElement(TDRecord, {t: "TXT Records", arr: urldata.xfe.resolve.TXT}),
                      React.createElement(TDRecord, {t: "MX Records", arr: urldata.xfe.resolve.MX, m: mxToStr})
                    )
                  ),
                  React.createElement(URLMalware, {urldata: urldata})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var URLRiskScore = React.createClass({displayName: "URLRiskScore",
      render: function() {
        var xfedata = this.props.data;
        if (xfedata.not_found ) {
          return null;
        }
        else {
          return (
            React.createElement("h3", null, " Risk Score: ", xfedata.url_details.score, " ")
          );
        }
      }
    });

    var TDRecord = React.createClass({displayName: "TDRecord",
      render: function() {
        var handleArr = function(arr, m) {
          if (arr && arr.length > 0) {
            if (m) {
              return arr.map(m).join(', ');
            }
            return arr.join(', ');
          }
          return '';
        }
        var data = handleArr(this.props.arr, this.props.m);
        if (data) {
          return(
            React.createElement("tr", null, React.createElement("td", null, this.props.t), React.createElement("td", null, data))
          );
        }
        return null;
      }
    });

    var URLCategory = React.createClass({displayName: "URLCategory",
      render: function() {
        var urldata = this.props.urldata;
        if (!urldata.xfe.not_found) {
          var categories = Object.keys(mapOrUnknown(urldata.xfe.url_details.cats)).join(', ');
          if (categories) {
            return (
              React.createElement("h3", null, "Categories: ", categories)
            );
          }
        }
        return null;
      }
    });

    var URLMalware = React.createClass({displayName: "URLMalware",
      render: function() {
        var malware = this.props.urldata.xfe.url_malware;
        if (malware && malware.count > 0) {
          var sortf = function(a, b) {
            return a.firstseen > b.firstseen ? -1 : b.firstseen > a.firstseen ? 1 : 0;
          }
          var sorted = malware.malware;
          sorted.sort(sortf);
          var rows = [];
          for (var i=0; i < sorted.length && i < 10; i++) {
            rows.push(React.createElement("tr", {key: 'mal_' + i}, React.createElement("td", null, sorted[i].firstseen), React.createElement("td", null, sorted[i].type), React.createElement("td", null, sorted[i].md5), React.createElement("td", null, sorted[i].uri), React.createElement("td", null, arrOrUnknown(sorted[i].family).join(', '))));
          }
          return (
            React.createElement("div", null,
              React.createElement("h3", null, "Malware detected on URL"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null, React.createElement("th", null, "First Seen"), React.createElement("th", null, "Type"), React.createElement("th", null, "MD5"), React.createElement("th", null, "URL"), React.createElement("th", null, "Family")),
                React.createElement("tbody", null, rows)
              )
            )
          );
        }
        return null;
      }
    });

    var URLVT = React.createClass({displayName: "URLVT",
      render: function() {
        var urldata = this.props.urldata;
        if (urldata && urldata.vt && urldata.vt.url_report && urldata.vt.url_report.response_code === 1) {
          var xfeFound = urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A);
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hurlvt"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {className: xfeFound ? 'collapsed' : '', role: "button", "data-toggle": "collapse", "data-parent": "#urlproviders", href: "#urlvt", "aria-expanded": "true", "aria-controls": "urlvt"},
                    "Virus Total Data"
                  )
                )
              ),
              React.createElement("div", {id: "urlvt", className: xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in', role: "tabpanel", "aria-labelledby": "hurlvt"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement("h4", null,
                    "Scan Date: ", urldata.vt.url_report.scan_date,
                    React.createElement("br", null),
                    "Positives: ", urldata.vt.url_report.positives,
                    React.createElement("br", null),
                    "Total: ", urldata.vt.url_report.total
                  ),
                  React.createElement(DetectedEngines, {urldata: urldata})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var DetectedEngines = React.createClass({displayName: "DetectedEngines",
      render: function() {
        var urldata = this.props.urldata;
        var detected_engines = [];
        var scans = urldata.vt.url_report.scans;
        for (var k in scans) {
          if (scans.hasOwnProperty(k) && scans[k].detected) {
            detected_engines.push(React.createElement("tr", {key: 'url_' + k}, React.createElement("td", null, k), React.createElement("td", null, scans[k].result)));
          }
        }
        if (detected_engines.length > 0) {
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Positive Detections"),
              React.createElement("table", null,
                React.createElement("thead", null, React.createElement("th", null, "Scan Engine"), React.createElement("th", null, "Result")),
                React.createElement("tbody", null, detected_engines)
              )
            )
          );
        }
        return null;
      }
    });
    // ======================== END URL section ===========================

    // ======================== File section ===========================
    var FileDiv = React.createClass({displayName: "FileDiv",
      render: function() {
        if (!isfile && !ismd5) {
          return null;
        } else {
          var data = this.props.data;
          return (
            React.createElement("div", null,
              React.createElement("h2", null, "File: ", isfile ? data.file.details.name : data.hashes[0].details),
              React.createElement(FileResultMessage, {data: data}),
              React.createElement(FileAV, {data: data}),
              React.createElement(FileDetails, {data: data.hashes[0]})
            )
          );
        }
      }
    });

    var FileResultMessage = React.createClass({displayName: "FileResultMessage",
      render: function() {
        var resultMessage = 'Could not determine the File reputation.';
        var color = 'warning-text';
        var filedata = this.props.data.file;
        var md5data = this.props.data.hashes[0];
        if (isfile && filedata.Result == 0 || ismd5 && md5data.Result == 0) {
          resultMessage = 'File is found to be clean.';
          color = 'success-text';
        } else if (isfile && filedata.Result == 1 || ismd5 && md5data.Result == 1) {
          resultMessage = 'File is found to be malicious.';
          color = 'danger-text';
        }
        return (React.createElement("h3", {className: color}, resultMessage));
      }
    });

    var FileAV = React.createClass({displayName: "FileAV",
      render: function() {
        var data = this.props.data;
        if (data && data.file && data.file.virus) {
          return (React.createElement("h2", null, " Malware Name: ", data.file.virus, " "));
        }
        return null;
      }
    });

    var FileDetails = React.createClass({displayName: "FileDetails",
      render: function() {
        var data = this.props.data;
        if (data &&
          (data.xfe && !data.xfe.not_found ||
          data.vt.file_report && data.vt.file_report.response_code === 1)) {
          return (
            React.createElement("div", {className: "panel-group", id: "fileproviders", role: "tablist", "aria-multiselectable": "true"},
              React.createElement(FileXFE, {data: data}),
              React.createElement(FileVT, {data: data})
            )
          );
        }
        return null;
      }
    });

    var FileXFE = React.createClass({displayName: "FileXFE",
      render: function() {
        var data = this.props.data;
        if (data && data.xfe && !data.xfe.not_found) {
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hfilexfe"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {role: "button", "data-toggle": "collapse", "data-parent": "#fileproviders", href: "#filexfe", "aria-expanded": "true", "aria-controls": "filexfe"},
                    "IBM X-Force Exchange Data"
                  )
                )
              ),
              React.createElement("div", {id: "filexfe", className: "panel-collapse collapse in", role: "tabpanel", "aria-labelledby": "hfilexfe"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement("table", {className: "table"},
                    React.createElement("tbody", null,
                      React.createElement("tr", null, React.createElement("td", null, "Type"), React.createElement("td", null, data.xfe.malware.type)),
                      React.createElement("tr", null, React.createElement("td", null, "Mime Type"), React.createElement("td", null, data.xfe.malware.mimetype)),
                      React.createElement("tr", null, React.createElement("td", null, "MD5"), React.createElement("td", null, data.xfe.malware.md5)),
                      React.createElement("tr", null, React.createElement("td", null, "Family"), React.createElement("td", null, arrOrUnknown(data.xfe.malware.family).join(', '))),
                      React.createElement("tr", null, React.createElement("td", null, "Created"), React.createElement("td", null, data.xfe.malware.created))
                    )
                  ),
                  React.createElement(FileXFEOriginsEmail, {data: data.xfe.malware.origins.emails.rows}),
                  React.createElement(FileXFEOriginsSubject, {data: data.xfe.malware.origins.subjects.rows}),
                  React.createElement(FileXFEOriginsDown, {data: data.xfe.malware.origins.downloadServers.rows}),
                  React.createElement(FileXFEOriginsCNC, {data: data.xfe.malware.origins.CnCServers.rows}),
                  React.createElement(FileXFEOriginsExt, {data: data.xfe.malware.origins.external})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var FileXFEOriginsEmail = React.createClass({displayName: "FileXFEOriginsEmail",
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'file_email' + i},
                React.createElement("td", null, data[i].firstseen), React.createElement("td", null, data[i].lastseen), React.createElement("td", null, data[i].origin), React.createElement("td", null, data[i].md5), React.createElement("td", null, data[i].filepath)
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Email Origins"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null, React.createElement("th", null, "First Seen"), React.createElement("th", null, "Last Seen"), React.createElement("th", null, "Origin"), React.createElement("th", null, "MD5"), React.createElement("th", null, "File Path")),
                React.createElement("tbody", null, rows)
              )
            )
          );
        }
        return null;
      }
    });

    var FileXFEOriginsSubject = React.createClass({displayName: "FileXFEOriginsSubject",
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'file_subject' + i},
                React.createElement("td", null, data[i].firstseen), React.createElement("td", null, data[i].lastseen), React.createElement("td", null, data[i].subject), React.createElement("td", null, data[i].ips ? data[i].ips.join(', ') : '')
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Subjects"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null, React.createElement("th", null, "First Seen"), React.createElement("th", null, "Last Seen"), React.createElement("th", null, "Subject"), React.createElement("th", null, "IPs")),
                React.createElement("tbody", null, rows)
              )
            )
          );
        }
        return null;
      }
    });

    var FileXFEOriginsDown = React.createClass({displayName: "FileXFEOriginsDown",
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'file_down' + i},
                React.createElement("td", null, data[i].firstseen), React.createElement("td", null, data[i].lastseen), React.createElement("td", null, data[i].host), React.createElement("td", null, data[i].uri)
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Download Servers"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null, React.createElement("th", null, "First Seen"), React.createElement("th", null, "Last Seen"), React.createElement("th", null, "Host"), React.createElement("th", null, "URI")),
                React.createElement("tbody", null, rows)
              )
            )
          );
        }
        return null;
      }
    });

    var FileXFEOriginsCNC = React.createClass({displayName: "FileXFEOriginsCNC",
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              React.createElement("tr", {key: 'file_cnc' + i},
                React.createElement("td", null, data[i].firstseen), React.createElement("td", null, data[i].lastseen), React.createElement("td", null, data[i].ip), React.createElement("td", null, arrOrUnknown(data[i].family).join(', '))
              )
            )
          }
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "Command & Control Servers"),
              React.createElement("table", {className: "table"},
                React.createElement("thead", null, React.createElement("th", null, "First Seen"), React.createElement("th", null, "Last Seen"), React.createElement("th", null, "IP"), React.createElement("th", null, "Family")),
                React.createElement("tbody", null, rows)
              )
            )
          );
        }
        return null;
      }
    });

    var FileXFEOriginsExt = React.createClass({displayName: "FileXFEOriginsExt",
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.family && data.family.length > 0) {
          return (
            React.createElement("div", null,
              React.createElement("h4", null, "External Detection"),
              React.createElement("h5", null, data.family.join(', '))
            )
          );
        }
        return null;
      }
    });

    var FileVT = React.createClass({displayName: "FileVT",
      render: function() {
        var data = this.props.data;
        if (data && data.vt && data.vt.file_report && data.vt.file_report.response_code === 1) {
          var xfeFound = data.xfe && !data.xfe.not_found;
          return (
            React.createElement("div", {className: "panel panel-default"},
              React.createElement("div", {className: "panel-heading", role: "tab", id: "hfilevt"},
                React.createElement("h4", {className: "panel-title"},
                  React.createElement("a", {className: xfeFound ? 'collapsed' : '', role: "button", "data-toggle": "collapse", "data-parent": "#fileproviders", href: "#filevt", "aria-expanded": "true", "aria-controls": "filevt"},
                    "Virus Total Data"
                  )
                )
              ),
              React.createElement("div", {id: "filevt", className: xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in', role: "tabpanel", "aria-labelledby": "hfilevt"},
                React.createElement("div", {className: "panel-body"},
                  React.createElement("h4", null,
                    "Scan Date: ", data.vt.file_report.scan_date,
                    React.createElement("br", null),
                    "Positives: ", data.vt.file_report.positives,
                    React.createElement("br", null),
                    "Total: ", data.vt.file_report.total
                  ),
                  React.createElement("table", {className: "table"}, React.createElement("tbody", null,
                    React.createElement("tr", null, React.createElement("td", null, "MD5"), React.createElement("td", null, data.vt.file_report.md5)),
                    React.createElement("tr", null, React.createElement("td", null, "SHA1"), React.createElement("td", null, data.vt.file_report.sha1)),
                    React.createElement("tr", null, React.createElement("td", null, "SHA256"), React.createElement("td", null, data.vt.file_report.sha256))
                  )),
                  React.createElement(ScanResult, {data: data.vt.file_report})
                )
              )
            )
          );
        }
        return null;
      }
    });

    var ScanResult = React.createClass ({displayName: "ScanResult",
      render: function() {
        var scans = this.props.data.scans;
        var rows = [];
        for (var k in scans) {
          if (scans.hasOwnProperty(k) && scans[k].detected) {
            rows.push(React.createElement("tr", {key: 'fvt_scan_' + k}, React.createElement("td", null, k), React.createElement("td", null, scans[k].result), React.createElement("td", null, scans[k].version), React.createElement("td", null, scans[k].update)));
          }
        }
        if (rows.length > 0) {
          return (
          React.createElement("div", null,
            React.createElement("h4", null, "Detection Engines"),
            React.createElement("table", {className: "table"},
              React.createElement("thead", null,
                React.createElement("th", null, "Engine Name"),
                React.createElement("th", null, "Version"),
                React.createElement("th", null, "Result"),
                React.createElement("th", null, "Update")
              ),
              React.createElement("tbody", null, rows)
            )
          )
          );
        }
        return null;
      }
    });

    var DetailsDiv = React.createClass({displayName: "DetailsDiv",
      loadDataFromServer: function() {
        $.ajax({
          type: 'GET',
          url: '/work',
          data: qParts,
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(data) {
            isfile = (data.type & FILEMask)  > 0;
            ismd5 = (data.type & MD5Mask)  > 0;
            isurl = (data.type & URLMask)  > 0;
            isip = (data.type & IPMask)  > 0;
            this.setState({data: data});
            this.setState({status: 1});
          }.bind(this),
          error: function(xhr, status, error) {
            // display the error on the UI
            this.setState({status: 2});
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              err += " - " + xhr.responseJSON.errors[0].detail;
              this.setState({errmsg: err});
            }
          }.bind(this)
        });
      },
      getInitialState: function() {
        return {data: [], status: 0};
      },
      componentDidMount: function() {
        this.loadDataFromServer();
      },
      render: function() {
            if (this.state.status == 0) {
              return(
                React.createElement("div", null,
                  React.createElement("h2", null, React.createElement("center", null, "D", React.createElement("small", null, "BOT"), " is collecting security details for your query. It might take up to a minute!")),
                  React.createElement("br", null),
                  React.createElement("br", null),
                  React.createElement("div", {className: "ball-grid-pulse center-block"},
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null),
                    React.createElement("div", null)
                  )

                )
              );
            } else if (this.state.status == 1) {
              return (
                React.createElement("div", null,
                  React.createElement("h1", {className: "text-center"}, "D", React.createElement("small", null, "BOT"), " Analysis Report"),
                  React.createElement(URLDiv, {data: this.state.data}),
                  React.createElement(FileDiv, {data: this.state.data}),
                  React.createElement(IPDiv, {data: this.state.data})
                )
              );
            }
            else {
              return(
                React.createElement("div", null,
                  "D", React.createElement("small", null, "BOT"), " encountered an error while trying to serve your request. The issues has been reported and will be analyzed." + ' ' +
                  "Please try to click the link again from Slack interface.",
                  React.createElement("hr", null),
                  this.state.errmsg
                )
              );
            }
        }
    });

    React.render(React.createElement(DetailsDiv, null), document.getElementById('detailsdiv'));
  }
})(window.jQuery);

// END Details Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#next').length) {
    var regexChannelsMatched = [];
    var channelsMatched = [];
    var groupsMatched = [];
    var allchecked = false;
    var im = false;


    var updateChannelList = function() {
      // update the channels Monitored
      var mergedList = new Object();
      var mergedArr = [];

      for (var i = 0; channelsMatched && i<channelsMatched.length; i++) {
        mergedList[channelsMatched[i]] = true;
      }
      for (var i = 0; groupsMatched && i<groupsMatched.length; i++) {
        mergedList[groupsMatched[i]] = true;
      }
      for (var i = 0; regexChannelsMatched && i<regexChannelsMatched.length; i++) {
        mergedList[regexChannelsMatched[i]] = true;
      }

      for (var k in mergedList) {
        mergedArr.push(k);
      }

      if (allchecked) {
        $('#channellist').html("<h2>All conversations are now being monitored.</h2>");
      } else if (mergedArr.length > 0) {
        $('#channellist').html("<h2>Channels Monitored</h2>");
        $('#channellist').append(mergedArr.sort().join(", "));
      } else {
        $('#channellist').html("<p class='warning-text'>D<small>BOT</small> is not monitoring any conversations. Please <b>select channels</b> to monitor below\
         or select <b>\'Monitor ALL conversations\'</b> above.</p>");
      }


    }


    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      $('.ball-grid-pulse').hide();

      for (var i=0; data.channels && i<data.channels.length; i++) {
        if (data.channels[i].selected)
          channelsMatched.push(data.channels[i].name)
      }

      for (var i=0; data.groups && i<data.groups.length; i++) {
        if (data.groups[i].selected)
          groupsMatched.push(data.groups[i].name)
      }

      allchecked = data.all;
      im = data.im;

      if (data.regexp) {
        $.ajax({
          type: 'POST',
          url: '/match',
          data: JSON.stringify({regexp: data.regexp}),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(data){
            // $('#regexpChannels').html('Channels Monitored: ' + data.join(', '));
            regexChannelsMatched = data;
            updateChannelList();
          }
        });
      }

      updateChannelList();


    });

    $.getJSON('/user', function(data) {

      $('#realname').html(data.real_name);
      $('#useremail').html(data.email);
      $('#teamname').html('Team: ' + data.team_name);

    });

  }
})(window.jQuery);


// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // load the feedback widget only on conf page
  $(function () {
    if ($('#configdiv').length) {
      // get the user information
      $.getJSON('/user', function(data) {
        $('#realname').text(data.real_name);
        $('#useremail').text(data.email);
        $('#teamname').text('Team: ' + data.team_name);
        zE(function() {
          zE.identify({
            name: data.real_name,
            email: data.email
          });
        });
      })
      .error( function(xhr, status, error) {
        var err = error;
        if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
          err += " - " + xhr.responseJSON.errors[0].detail;
          if (xhr.responseJSON.errors[0].status == 401) {
            // unauthorized user - user not logged in
            $('#unauthmodal').modal('show');
            window.setTimeout(function() {
              $('#unauthmodal').modal('hide');
              window.location.replace("/");
            }, 3000);
          }
        }
      });
    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
