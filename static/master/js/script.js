/*!
 *
 * Alfred - Demisto Security Bot
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
  var stickySelector = '.navbar-sticky';

  // Setup functions based on screen
  if (matchMedia('(min-width: 992px), (max-width: 767px)').matches) {
    stickyNavScroll = function () {
      var top = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
      if (top > 40) $(stickySelector).stop().animate({'top': '0'});

      else $(stickySelector).stop().animate({'top': '-80'});
    };
  }

  if (matchMedia('(min-width: 768px) and (max-width: 991px)').matches) {
    stickyNavScroll = function () {
      var top = (document.documentElement && document.documentElement.scrollTop) || document.body.scrollTop;
      if (top > 40) $(stickySelector).stop().animate({'top': '0'});

      else $(stickySelector).stop().animate({'top': '-120'});
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

  // $(function() {
  //
  //   if (matchMedia('(min-width: 640px)').matches) {
  //
  //     var videobackground = new $.backgroundVideo( $body, {
  //       'align':    'centerXY',
  //       'width':    1280,
  //       'height':   720,
  //       'path':     'video/',
  //       'filename': 'video',
  //       'types':    ['mp4', 'webm']
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


  // Nivo Lightbox
  // -----------------------------------
  $(function () {

    $('#appshots a').nivoLightbox({

      effect: 'fadeScale',                        // The effect to use when showing the lightbox
      theme: 'default',                           // The lightbox theme to use
      keyboardNav: true                           // Enable/Disable keyboard navigation (left/right/escape)

    });

  });

  // Check if we already have the cookie and if so, change the title of the button
  // -----------------------------------
  $(function () {
    // If we are on the homepage
    if ($('#slack-message').length) {
      $.getJSON('/user', function(data) {
        $('#slack-message').text('Configure Alfred');
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
