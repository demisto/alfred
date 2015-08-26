
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#channels').length) {
    var timerExists = false;
    var regexChannelsMatched = [];
    var channelsMatched = [];
    var groupsMatched = [];

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

      if (mergedArr.length > 0) {
        $('#channellist').html(mergedArr.sort().join(", "));
      } else {
        $('#channellist').html("<p class='warning-text'>DBOT is not monitoring any conversations. Please <b>select channels</b> to monitor below\
         or select <b>\'Monitor ALL conversations\'</b> above.</p>");
      }


    }


    var disableAll = function() {

      $("#channels").attr("disabled", true).trigger("chosen:updated");
      $("#groups").attr("disabled", true).trigger("chosen:updated");
      $("#im").attr("disabled", true);
      $("#verbosechannels").attr("disabled", true).trigger("chosen:updated");
      $("#verbosegroups").attr("disabled", true).trigger("chosen:updated");
      $("#verboseim").attr("disabled", true);
      $('#headingConf').addClass('grayout');
      $('#configpanel').addClass('grayout');
      $('#headingConf a').removeAttr("href");
      $('#verboseheadingConf').addClass('grayout');
      $('#verboseconfigpanel').addClass('grayout');
      $('#verboseheadingConf a').removeAttr("href");
      $('#verboseheadingAdvConf .collapsed').removeAttr("href");
      $('#channelsmonitored').hide();


    };
    var enableAll = function() {
      $("#channels").removeAttr("disabled").trigger("chosen:updated");
      $("#groups").removeAttr("disabled").trigger("chosen:updated");
      $("#im").removeAttr("disabled");
      $("#verbosechannels").removeAttr("disabled").trigger("chosen:updated");
      $("#verbosegroups").removeAttr("disabled").trigger("chosen:updated");
      $("#verboseim").removeAttr("disabled");
      $('#headingConf').removeClass('grayout');
      $('#configpanel').removeClass('grayout');
      $('#headingConf a').attr("href", "#configpanel");
      $('#verboseheadingConf').removeClass('grayout');
      $('#verboseconfigpanel').removeClass('grayout');
      $('#verboseheadingAdvConf .collapsed').attr("href", '#verboseconfig');
      $('#channelsmonitored').show();
    }


    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      $('.ball-grid-pulse').hide();
      $('#configdiv').show();

      var channels = [];
      var verbosechannels = [];
      var groups = [];
      var verbosegroups = [];

      for (var i=0; data.channels && i<data.channels.length; i++) {
        if (data.channels[i].verbose) {
          verbosechannels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].selected ? 'selected' : '') + '>' + data.channels[i].name + '</option>');
        } else {
          channels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].selected ? 'selected' : '') + '>' + data.channels[i].name + '</option>');
        }

        if (data.channels[i].selected)
          channelsMatched.push(data.channels[i].name)
      }
      $('#channels').append(channels.join(''));
      $('#verbosechannels').append(verbosechannels.join(''));

      for (var i=0; data.groups && i<data.groups.length; i++) {
        if (data.groups[i].verbose) {
          verbosegroups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].selected ? 'selected' : '') + '>' + data.groups[i].name + '</option>');
        } else {
          groups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].selected ? 'selected' : '') + '>' + data.groups[i].name + '</option>');
        }

        if (data.groups[i].selected)
          groupsMatched.push(data.groups[i].name)
      }

      $('#groups').append(groups.join(''));
      $('#verbosegroups').append(groups.join(''));

      $('#im').prop('checked', data.im);
      $('#verboseim').prop('checked', data.verbose_im);
      $('#all').prop('checked', data.all);
    

      updateChannelList();

      // If all is enabled then disable all the others
      if (data.all) {
        disableAll();
      }

      // Enable chosen
      $(".chosen-select").chosen({no_results_text: "Oops, No matching entry found:"});

      // Function to save all properties
      var saveAll = function() {
        var save = {};
        save.channels = [];
        save.groups = [];
        save.im = $('#im').is(':checked');
        save.all = $('#all').is(':checked');
        save.regexp = $('#regexp').val();
        channelsMatched = [];
        groupsMatched = [];
        $('#channels option:selected').each(function() {
          save.channels.push($(this).val());
          channelsMatched.push($(this).text());
        });
        $('#groups option:selected').each(function() {
          save.groups.push($(this).val());
          groupsMatched.push($(this).text());
        });

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
              "positionClass": "toast-top-center",
              "preventDuplicates": true,
              "onclick": null,
              "showDuration": "300",
              "hideDuration": "1000",
              "timeOut": "5000",
              "extendedTimeOut": "1000",
              "showEasing": "swing",
              "hideEasing": "linear",
              "showMethod": "fadeIn",
              "hideMethod": "fadeOut"
            };
            toastr["success"]("Configuration Saved");
          },
          error: function(xhr, status, error) {
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              err += " - " + xhr.responseJSON.errors[0].detail;
            }
            $('#regexpChannels').html(err);
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
      $('.chosen-select,#im').change(function(evt) {
        saveAll();
      });

      var saveRegex = function(evt)
      {
        var timer;
        if (evt.type == "keyup") {
          if (!timerExists)
          {
            timerExists = true;
            var timer = setTimeout(function(){
              timerExists = false;
              if (evt && evt.target && evt.target.value) {
                $.ajax({
                  type: 'POST',
                  url: '/match',
                  data: JSON.stringify({regexp: evt.target.value}),
                  headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
                  dataType: 'json',
                  contentType: 'application/json; charset=utf-8',
                  success: function(data){
                    regexChannelsMatched = data;
                    // $('#regexpChannels').html('Matched Channels: ' + data.join(', '));
                    saveAll();
                  },
                  error: function(xhr, status, error) {
                    var err = error;
                    if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
                      err += " - " + xhr.responseJSON.errors[0].detail;
                    }
                    $('#regexpChannels').html(err);
                  }
                });
              }
              else if (evt && evt.target) {
                regexChannelsMatched = [];
                saveAll();
              }
            }, 5000);
          }
        }
        else if (evt.type == "change") {
          if (timerExists) {
            clearTimeout(timer);
            timerExists = false;
          }
          if (evt && evt.target && evt.target.value) {
            $.ajax({
              type: 'POST',
              url: '/match',
              data: JSON.stringify({regexp: evt.target.value}),
              headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
              dataType: 'json',
              contentType: 'application/json; charset=utf-8',
              success: function(data){
                regexChannelsMatched = data;
                saveAll();
              },
              error: function(xhr, status, error) {
                var err = error;
                if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
                  err += " - " + xhr.responseJSON.errors[0].detail;
                }
                $('#regexpChannels').html(err);
              }
            });
          }
        }
      }


      $('#regexp').keyup(saveRegex);
      $('#regexp').change(saveRegex);

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
