
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';

  var updatePage = function(jsonResult) {
//    $('#forfile').text(jsonResult.file.details.name);

    var MD5Mask = 1;
    var URLMask = 2;
    var IPMask = 4;
    var FILEMask = 8;

    var isfile = (jsonResult.type & FILEMask)  > 0;
    var ismd5 = (jsonResult.type & MD5Mask)  > 0;
    var isurl = (jsonResult.type & URLMask)  > 0;
    var isip = (jsonResult.type & IPMask)  > 0;

    if (!isurl) {
      $('#url').hide();
    }
    else {
      // fill in the details for the url
      $('#forurl').text(jsonResult.url.xfe.url_details.url);

      if (jsonResult.url.xfe.resolve.A != null && jsonResult.url.xfe.resolve.A.length > 0) {
        $('#arecords').text(jsonResult.url.xfe.resolve.A);
      }
      else {
        $('#asection').hide();
      }

      if (jsonResult.url.xfe.resolve.AAAA != null && jsonResult.url.xfe.resolve.AAAA.length > 0) {
        $('#ipv6rec').text(jsonResult.url.xfe.resolve.AAAA);
      }
      else {
        $('#ipv6section').hide();
      }


      if (jsonResult.url.xfe.resolve.TXT != null && jsonResult.url.xfe.resolve.TXT.length > 0) {
        $('#txtrec').text(jsonResult.url.xfe.resolve.TXT);
      }
      else {
        $('#txtsection').hide();
      }

      if (jsonResult.url.xfe.resolve.MX != null && jsonResult.url.xfe.resolve.MX.length > 0) {
        //TODO : should we display this better - we get an array of records. Where is the doc for this?
        $('#mxrec').text(JSON.stringify(jsonResult.url.xfe.resolve.MX));
      }
      else {
        $('#mxsection').hide();
      }

      var cats = jsonResult.url.xfe.url_details.cats;
      var catfound = false;
      for (var k in cats) {
        if (cats[k]) {
          catfound = true;
          $('#cat').text(k);
        }
      }
      if (!catfound) {
        $('#catsection').hide();
      }

      // display the engines that convicted this URL
      if (jsonResult.url.vt.url_report.positives > 0) {
        var numEngines = jsonResult.url.vt.url_report.scans.length;
        var scanMap = jsonResult.url.vt.url_report.scans;
        for (var k in scanMap) {
          if (scanMap[k].detected) {
            $('#detectedEngines').append(k);
          }
        }
      }

    }
    if (!isip) {
      $('#ip').hide();
    }
    else {
      // fill in the details for IP section
      $('#forip').text(jsonResult.ip.xfe.ip_reputation.ip);

      // TODO: why is subnets an array?
      //      $('#subnetdata').text(jsonResult.ip.xfe.ip_reputation.subnets.subnet);

      // Geo Data
      if (jsonResult.ip.xfe.ip_reputation.geo != null) {
        $('#geodata').text(jsonResult.ip.xfe.ip_reputation.geo['country']);
      }
      //Historical resolutions
      var resArray = jsonResult.ip.VT.ip_report.Resolutions;
      if (resArray != null) {
        var numResolutions = resArray.length;
        if (numResolutions == 0)
          $('#ressection').hide();
        for (var i = 0; i < numResolutions; i++) {
          $('#resolvetable').append('<tr><td>'+ resArray[i].hostname + '</td><td>' + resArray[i].last_resolved + '</td></tr>');
        }
      }
      else
        $('#ressection').hide();


      // some of the resolved URLs are detected as malicious
      var detectedURLArr = jsonResult.ip.VT.ip_report.detected_urls;
      if (detectedURLArr != null) {
        var detectedURLArrLen = detectedURLArr.length;
        if (detectedURLArrLen == 0)
          $('#detected_urls_section').hide();
        for (var i=0; i < detectedURLArrLen; i++) {
          $('#detected_urls_table').append('<tr><td>'+ detectedURLArr[i].url + '</td><td>'
            + detectedURLArr[i].positives + '</td><td>' + detectedURLArr[i].scan_date + '</td></tr>');
        }
      }
      else {
        $('#detected_urls_section').hide();
      }

    }
    if (!isfile && !ismd5) {
      $('#file').hide();
    } else {
      // either the md5 or file is present.
      // when file is present then we will show the AV result and the md5 from VT and XFE.
      //if only md5 present then we do not show AV

    }

//    $('#forip').text(jsonResult.ip.);


  };


  // Run this only on details page
  if ($('#details').length) {
    var uri = new URI();
    var qParts = uri.search(true);

    $.ajax({
      type: 'GET',
      url: '/work',
      data: qParts,
      headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
      dataType: 'json',
      contentType: 'application/json; charset=utf-8',
      success: updatePage
    });

      }
    })(window.jQuery);

// END Settings Handler
// -----------------------------------
