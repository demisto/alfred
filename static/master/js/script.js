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



  $win.load(function() {
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


  // Wow Animation
  // -----------------------------------

  // setup global config
  window.wow = (
      new WOW({
      mobile: false
    })
  ).init();

  // Check if we already have the cookie and if so, change the title of the button
  // -----------------------------------
  $(function () {
    // If we are on the homepage
    if ($('#counter').length) {
      var counter_timer;

      // counter
        var options = {
          useEasing : false,
          useGrouping : true,
          separator : '',
          decimal : '.',
          prefix : '',
          suffix : ''
        };
       var counter = new CountUp("counter", 500, 10000, 0, 30, options);
       counter.start();
      //
      var last_stop_counter;
      var refreshCounter = function(start_count, stop_count, duration) {
        var options = {
          useEasing : false,
          useGrouping : true,
          separator : '|',
          decimal : '.',
          prefix : '',
          suffix : ''
        };
      //  var counter = new CountUp("counter", start_count, stop_count, 0, duration, options);
      //  last_stop_counter = stop_count;
      //  counter.start();
      }

      $.getJSON('/messages', function(data) {
        var s_count = data.count - 100;
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
        $('#action').attr('href', '/conf')
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
