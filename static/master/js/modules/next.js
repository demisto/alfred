(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#next').length) {
    var regexChannelsMatched = [];
    var channelsMatched = [];
    var groupsMatched = [];
    var allchecked = false;
    var im = false;
    $('#closelink').hide();

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
        $('#channellist').html("<p class='warning-text'>DBOT is not monitoring any conversations. Please <b>select channels</b> to monitor below\
         or select <b>\'Monitor ALL conversations\'</b> above.</p>");
      }


    }


    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      $('.ball-grid-pulse').hide();
      $('#closelink').show();

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
