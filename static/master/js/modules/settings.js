
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#channels').length) {
    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      var channels = [];
      var groups = [];
      for (var i=0; data.channels && i<data.channels.length; i++) {
        channels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].selected ? 'selected' : '') + '>' + data.channels[i].name + '</option>');
      }
      $('#channels').append(channels.join(''));
      for (var i=0; data.groups && i<data.groups.length; i++) {
        groups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].selected ? 'selected' : '') + '>' + data.groups[i].name + '</option>');
      }
      $('#groups').append(groups.join(''));
      $('#im').prop('checked', data.im);
      // Enable chosen
      $(".chosen-select").chosen({no_results_text: "Oops, No matching entry found:"});
      $('.chosen-select,#im').change(function(evt, params) {
        var save = {};
        save.channels = [];
        save.groups = [];
        save.im = $('#im').is(':checked');
        $('#channels option:selected').each(function() {
          save.channels.push($(this).val());
        });
        $('#groups option:selected').each(function() {
          save.groups.push($(this).val());
        });
        $.ajax({
          type: 'POST',
          url: '/save',
          data: JSON.stringify(save),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(){
            // TODO - clear the toaster
          }
        });
      });
    });
  }

  // load the feedback widget only on conf page
  $(function (){
    if ($('#channels').length) {
      FreshWidget.init("", {"queryString": "&widgetType=popup", "utf8": "✓",
        "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
        "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );
    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
