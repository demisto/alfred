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
  var features_section_top = $("#features").offset().top;
  var features_section_bottom = features_section_top + $("#features").outerHeight();

  // Setup functions based on screen
  if (matchMedia('(min-width: 992px), (max-width: 767px)').matches) {
    stickyNavScroll = function () {
      var top = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
      var social_widget_top = $("#social").offset().top;
      var social_widget_bottom = social_widget_top + $("#social").outerHeight();

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
  $doc.ready(stickyNavScroll);
  $win.scroll(stickyNavScroll);


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
  // $(function() {
  //
  //   if (matchMedia('(min-width: 640px)').matches) {
  //
  //     var videobackground = new $.backgroundVideo( $body, {
  //       'align':    'centerXY',
  //       'width':    1280,
  //       'height':   720,
  //       'path':     'img/videobg.mp4',
  //       'filename': 'video',
  //       'types':    ['mp4']
  //     });
  //   }
  //
  // });


  // Smooth Scroll
  // -----------------------------------
  var scrollAnimationTime = 1200,
      scrollAnimationFunc = 'easeInOutExpo',
      $root               = $('html, body');

  $(function(){
    $('.scrollto').on('click.smoothscroll', function (event) {

      event.preventDefault();

      var target = this.hash;

      // console.log($(target).offset().top)

      $root.stop().animate({
          'scrollTop': $(target).offset().top
      }, scrollAnimationTime, scrollAnimationFunc, function () {
          window.location.hash = target;
      });
    });

  });

  // Self close navbar on mobile click
  // -----------------------------------
  $(function(){
       var navMain = $("#navbar-main");
       var navToggle = $('.navbar-toggle');

       navMain.on('click', 'a', null, function () {
          if ( navToggle.is(':visible') )
            navMain.collapse('hide');
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


  // Owl Crousel
  // -----------------------------------

  $(function () {

    $('#feedback-carousel').owlCarousel({
        responsiveClass:  true,
        responsive: {
            0: {
                items: 1,
                nav:   false
            }
        }
    });

    $('#appshots').owlCarousel({
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

  });

  // Check if we already have the cookie and if so, change the title of the button
  // -----------------------------------
  $(function () {
    // If we are on the homepage
    if ($('#slack-message').length) {
      $.getJSON('/user', function(data) {
        $('#slack-message').text('Configure DBOT');
        $('#action').attr('href', '/conf')
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
