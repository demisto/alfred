
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#channels').length) {
    var channelsMatched = [];
    var groupsMatched = [];
    var verbosechannelsMatched = [];
    var verbosegroupsMatched = [];
    var allMonitored = false;


    var updateChannelList = function() {
      // update the channels Monitored
      var mergedList = {};
      var mergedArr = [];
      var verbosemergedList = {};
      var verbosemergedArr = [];

      for (var i = 0; channelsMatched && i<channelsMatched.length; i++) {
        mergedList[channelsMatched[i]] = true;
      }
      for (var i = 0; groupsMatched && i<groupsMatched.length; i++) {
        mergedList[groupsMatched[i]] = true;
      }
      for (var i = 0; verbosechannelsMatched && i<verbosechannelsMatched.length; i++) {
        verbosemergedList[verbosechannelsMatched[i]] = true;
      }
      for (var i = 0; verbosegroupsMatched && i<verbosegroupsMatched.length; i++) {
        verbosemergedList[verbosegroupsMatched[i]] = true;
      }

      for (var k in mergedList) {
        mergedArr.push(k);
      }

      for (var k in verbosemergedList) {
        verbosemergedArr.push(k);
      }

      $('#channellist').html('');
      $('#verbosechannellist').html('');
      if (allMonitored) {
        $('#channellist').html("DBOT is monitoring all conversations for your team. You can close the browser and get back to work.");
      } else if mergedArr.length > 0 {
        $('#channellist').append('<p>' + mergedArr.sort().join(", ")+'</p>');
      } else if (verbosemergedArr.length == 0){
        $('#channellist').html("<p class='warning-text'>DBOT is not monitoring any conversations. Please <b>select channels</b> to monitor below\
         or select <b>\'Monitor ALL conversations\'</b>.</p>");
      }
      if (verbosemergedArr.length > 0) {
        $('#verbosechannelsmonitored').show();
        $('#verbosechannellist').append('<p>' + verbosemergedArr.sort().join(", ")+'</p>');
      } else {
        $('#verbosechannelsmonitored').hide();
      }
    }

    var disableAll = function() {

      $("#channels").attr("disabled", true).trigger("chosen:updated");
      $("#groups").attr("disabled", true).trigger("chosen:updated");
      $("#im").attr("disabled", true);
      $('#headingConf').addClass('grayout');
      $('#configpanel').addClass('grayout');
      $('#headingConf a').removeAttr("href");


    };
    var enableAll = function() {
      $("#channels").removeAttr("disabled").trigger("chosen:updated");
      $("#groups").removeAttr("disabled").trigger("chosen:updated");
      $("#im").removeAttr("disabled");
      // $("#verbosechannels").removeAttr("disabled").trigger("chosen:updated");
      // $("#verbosegroups").removeAttr("disabled").trigger("chosen:updated");
      // $("#verboseim").removeAttr("disabled");
      $('#headingConf').removeClass('grayout');
      $('#configpanel').removeClass('grayout');
      $('#headingConf a').attr("href", "#configpanel");
      // $('#verboseheadingConf').removeClass('grayout');
      // $('#verboseconfigpanel').removeClass('grayout');
      // $('#verboseheadingConf a').attr("href", '#verboseconfigpanel');
      // $('#channelsmonitored').show();
    }


    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      $('.ball-grid-pulse').hide();
      $('#configdiv').show();
      $('#channelsmonitored').show();

      var channels = [];
      var verbosechannels = [];
      var groups = [];
      var verbosegroups = [];

      for (var i=0; data.channels && i<data.channels.length; i++) {

        verbosechannels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].verbose ? 'selected' : '') + '>' + data.channels[i].name + '</option>');
        channels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].selected ? 'selected' : '') + '>' + data.channels[i].name + '</option>');


        if (data.channels[i].selected)
          channelsMatched.push(data.channels[i].name)

        if (data.channels[i].verbose)
            verbosechannelsMatched.push(data.channels[i].name)

      }
      $('#channels').append(channels.join(''));
      $('#verbosechannels').append(verbosechannels.join(''));

      for (var i=0; data.groups && i<data.groups.length; i++) {
        verbosegroups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].verbose ? 'selected' : '') + '>' + data.groups[i].name + '</option>');
        groups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].selected ? 'selected' : '') + '>' + data.groups[i].name + '</option>');

        if (data.groups[i].selected)
          groupsMatched.push(data.groups[i].name)

        if (data.groups[i].verbose)
          verbosegroupsMatched.push(data.groups[i].name)

      }

      $('#groups').append(groups.join(''));
      $('#verbosegroups').append(verbosegroups.join(''));

      $('#im').prop('checked', data.im);
      $('#verboseim').prop('checked', data.verbose_im);
      $('#all').prop('checked', data.all);
      allMonitored = data.all;


      updateChannelList();

      // If all is enabled then disable all the others
      if (data.all) {
        disableAll();
      }

      // Enable chosen
      $(".chosen-select").chosen({no_results_text: "Oops, No matching entry found:"});
      $('#verboseconfigpanel').collapse('hide');

      // Function to save all properties
      var saveAll = function() {
        var save = {};
        save.channels = [];
        save.groups = [];
        save.im = $('#im').is(':checked');
        save.verbose_channels = [];
        save.verbose_groups = [];
        save.verbose_im = $('#verboseim').is(':checked');
        save.all = $('#all').is(':checked');
        channelsMatched = [];
        groupsMatched = [];
        verbosechannelsMatched = [];
        verbosegroupsMatched = [];

        $('#channels option:selected').each(function() {
          save.channels.push($(this).val());
          channelsMatched.push($(this).text());
        });
        $('#verbosechannels option:selected').each(function() {
          save.verbose_channels.push($(this).val());
          verbosechannelsMatched.push($(this).text());
        });
        $('#groups option:selected').each(function() {
          save.groups.push($(this).val());
          groupsMatched.push($(this).text());
        });
        $('#verbosegroups option:selected').each(function() {
          save.verbose_groups.push($(this).val());
          verbosegroupsMatched.push($(this).text());
        });
        allMonitored = $('#all').is(':checked');

        updateChannelList();

        $.ajax({
          type: 'POST',
          url: '/save',
          data: JSON.stringify(save),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(){
            toastr.options = {
              "closeButton": false,
              "debug": false,
              "newestOnTop": false,
              "progressBar": false,
              "positionClass": "toast-bottom-full-width",
              "preventDuplicates": true,
              "onclick": null,
              "showDuration": "300",
              "hideDuration": "1000",
              "timeOut": "3000",
              "extendedTimeOut": "1000",
              "showEasing": "swing",
              "hideEasing": "linear",
              "showMethod": "fadeIn",
              "hideMethod": "fadeOut"
            }
            toastr["success"]("Configuration saved.")
          },
          error: function(xhr, status, error) {
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              err += " - " + xhr.responseJSON.errors[0].detail;
              toastr.options = {
                "closeButton": false,
                "debug": false,
                "newestOnTop": false,
                "progressBar": false,
                "positionClass": "toast-bottom-full-width",
                "preventDuplicates": true,
                "onclick": null,
                "showDuration": "300",
                "hideDuration": "1000",
                "timeOut": "3000",
                "extendedTimeOut": "1000",
                "showEasing": "swing",
                "hideEasing": "linear",
                "showMethod": "fadeIn",
                "hideMethod": "fadeOut"
              }
              toastr["error"](err)
            }
          }
        });
      };
      $('#all').change(function(evt) {
        if ($('#all').is(':checked')) {
          disableAll();
        } else {
          enableAll();
        }
        saveAll();
      });
      $('.chosen-select,#im,#verboseim').change(function(evt) {
        saveAll();
      });



    });
  }

  // load the feedback widget only on conf page
  $(function (){
    if ($('#channels').length) {
      // get the user information
      $.getJSON('/user', function(data) {

        $('#realname').html(data.real_name);
        $('#useremail').html(data.email);
        $('#teamname').html('Team: ' + data.team_name);

        FreshWidget.init("", {"queryString": "&widgetType=popup&searchArea=no&helpdesk_ticket[subject]=Configuration:&helpdesk_ticket[requester]="+data.email, "utf8": "✓",
          "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
          "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );

      });

    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
