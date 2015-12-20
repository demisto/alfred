
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

    var IPDiv = React.createClass({
      render: function() {
        if (!isip) {
          return null;
        }
        else {
          var ipdata = this.props.data.ips[0];
          return (
            <div>
              <h2>IP: {ipdata.details}</h2>
              <IPResultMessage data={ipdata} />
              <IPDetails data={ipdata} />
            </div>
          );
        }
      }
    });

    var IPResultMessage = React.createClass({
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
        return (<h3 className={color}>{resultMessage}</h3>)
      }
    });

    var IPDetails = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data &&
          (data.xfe && !data.xfe.not_found ||
          data.vt.ip_report && data.vt.ip_report.response_code === 1)) {
          return (
            <div className="panel-group" id="ipproviders" role="tablist" aria-multiselectable="true">
              <IPXFE data={data} />
              <IPVT data={data} />
            </div>
          );
        }
        return null;
      }
    });

    var IPXFE = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data && data.xfe && !data.xfe.not_found) {
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hipxfe">
                <h4 className="panel-title">
                  <a role="button" data-toggle="collapse" data-parent="#ipproviders" href="#ipxfe" aria-expanded="true" aria-controls="ipxfe">
                    IBM X-Force Exchange Data
                  </a>
                </h4>
              </div>
              <div id="ipxfe" className="panel-collapse collapse in" role="tabpanel" aria-labelledby="hipxfe">
                <div className="panel-body">
                  <h3> Risk Score: {data.xfe.ip_reputation.score}</h3>
                  <h3> Country: {data.xfe.ip_reputation.geo && data.xfe.ip_reputation.geo['country'] ? data.xfe.ip_reputation.geo['country'] : 'Unknown'} </h3>
                  <h3> Categories: {Object.keys(mapOrUnknown(data.xfe.ip_reputation.cats)).join(', ')} </h3>
                  <SubnetSection data={data.xfe.ip_reputation.subnets} />
                  <IPHistory data={data.xfe.ip_history.history} />
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var SubnetSection = React.createClass({
      render: function() {
        var rows = [];
        var subnets = this.props.data;
        if (subnets && subnets.length > 0) {
          subnets.sort(sortByCreate);
          for (var i=0; i < subnets.length && i < 10; i++) {
            rows.push(
              <tr key={'ipr_subnet_' + i}>
                <td>{subnets[i].subnet}</td>
                <td>{subnets[i].score}</td>
                <td>{Object.keys(mapOrUnknown(subnets[i].cats)).join(', ')}</td>
                <td>{subnets[i].geo && subnets[i].geo['country'] ? subnets[i].geo['country'] : 'Unknown'}</td>
                <td>{subnets[i].reason}</td>
                <td>{subnets[i].created}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>Subnets</h4>
              <table className="table">
                <thead>
                  <th>Subnet</th>
                  <th>Score</th>
                  <th>Category</th>
                  <th>Location</th>
                  <th>Reason</th>
                  <th>Created</th>
                </thead>
                <tbody>
                  {rows}
                </tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var IPHistory = React.createClass({
      render: function() {
        var rows = [];
        var history = this.props.data;
        if (history && history.length > 0) {
          history.sort(sortByCreate);
          for (var i=0; i < history.length && i < 10; i++) {
            rows.push(
              <tr key={'ipr_hist_' + i}>
                <td>{history[i].ip}</td>
                <td>{history[i].score}</td>
                <td>{Object.keys(mapOrUnknown(history[i].cats)).join(', ')}</td>
                <td>{history[i].geo && history[i].geo['country'] ? history[i].geo['country'] : 'Unknown'}</td>
                <td>{history[i].reason}</td>
                <td>{history[i].created}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>IP History</h4>
              <table className="table">
                <thead>
                  <th>IP</th>
                  <th>Score</th>
                  <th>Category</th>
                  <th>Location</th>
                  <th>Reason</th>
                  <th>Created</th>
                </thead>
                <tbody>
                  {rows}
                </tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var IPVT = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data && data.vt && data.vt.ip_report && data.vt.ip_report.response_code === 1) {
          var xfeFound = data.xfe && !data.xfe.not_found;
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hurlvt">
                <h4 className="panel-title">
                  <a className={xfeFound ? 'collapsed' : ''} role="button" data-toggle="collapse" data-parent="#ipproviders" href="#ipvt" aria-expanded="true" aria-controls="ipvt">
                    Virus Total Data
                  </a>
                </h4>
              </div>
              <div id="ipvt" className={xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in'} role="tabpanel" aria-labelledby="hipvt">
                <div className="panel-body">
                  <ResolutionSection data={data.vt.ip_report.Resolutions}/>
                  <DetectedURLSection data={data.vt.ip_report.detected_urls}/>
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var DetectedURLSection = React.createClass({
      render: function() {
        var detected = this.props.data;
        if (detected && detected.length > 0) {
          detected.sort(sortByScanDate);
          var rows = [];
          for (var i=0; i < detected.length && i < 10; i++) {
            rows.push(<tr key={'ip_detected_' + i}><td>{detected[i].url}</td><td>{detected[i].positives} / {detected[i].total}</td><td>{detected[i].scan_date}</td></tr>);
          }
          return (
            <div>
              <h4>Detected URLs</h4>
              <table className="table">
                <thead>
                  <th style={{width:'70%'}}>URL</th>
                  <th>Positives</th>
                  <th>Scan Date</th>
                </thead>
                <tbody>
                  {rows}
                </tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var ResolutionSection = React.createClass({
      render: function() {
        var resArr = this.props.data;
        if (resArr && resArr.length > 0) {
          resArr.sort(sortByLastResolved);
          var rows = [];
          for (var i=0; i<resArr.length && i < 10; i++) {
            rows.push(<tr key={'resolv_' + i}><td>{resArr[i].hostname}</td><td>{resArr[i].last_resolved}</td></tr>);
          }
          return (
            <div>
              <h4>Historical Resolutions</h4>
              <table className="table">
                <thead>
                  <th>Hostname</th>
                  <th>Last Resolved</th>
                </thead>
                <tbody>
                  {rows}
                </tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });
    // ======================== END IP section ===========================

    // ======================== URL section ===========================
    var URLDiv = React.createClass({
      render: function() {
        if (!isurl) {
          return null;
        } else {
          var urldata = this.props.data.urls[0];
          return (
            <div>
              <h2>URL: {urldata.details}</h2>
              <URLResultMessage data={urldata} />
              <URLDetails urldata={urldata} />
            </div>
          );
        }
      }
    });

    var URLResultMessage = React.createClass({
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
        return (<h3 className={color}>{resultMessage}</h3>)
      }
    });

    var URLDetails = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        if (urldata &&
          (urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A) ||
          urldata.vt.url_report && urldata.vt.url_report.response_code === 1)) {
          return (
            <div className="panel-group" id="urlproviders" role="tablist" aria-multiselectable="true">
              <URLXFE urldata={urldata} />
              <URLVT urldata={urldata} />
            </div>
          );
        }
        return null;
      }
    });

    var URLXFE = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        if (urldata && urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A)) {
          var mxToStr = function(mx) {
            return mx.exchange + '(' + mx.priority + ')';
          }
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hurlxfe">
                <h4 className="panel-title">
                  <a role="button" data-toggle="collapse" data-parent="#urlproviders" href="#urlxfe" aria-expanded="true" aria-controls="urlxfe">
                    IBM X-Force Exchange Data
                  </a>
                </h4>
              </div>
              <div id="urlxfe" className="panel-collapse collapse in" role="tabpanel" aria-labelledby="hurlxfe">
                <div className="panel-body">
                  <URLRiskScore data={urldata.xfe} />
                  <URLCategory urldata={urldata} />
                  <table className="table">
                    <thead><th>Name</th><th>Value</th></thead>
                    <tbody>
                      <TDRecord t="A Records" arr={urldata.xfe.resolve.A} />
                      <TDRecord t="AAAA Records" arr={urldata.xfe.resolve.AAAA} />
                      <TDRecord t="TXT Records" arr={urldata.xfe.resolve.TXT} />
                      <TDRecord t="MX Records" arr={urldata.xfe.resolve.MX} m={mxToStr} />
                    </tbody>
                  </table>
                  <URLMalware urldata={urldata} />
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var URLRiskScore = React.createClass({
      render: function() {
        var xfedata = this.props.data;
        if (xfedata.not_found ) {
          return null;
        }
        else {
          return (
            <h3> Risk Score: {xfedata.url_details.score} </h3>
          );
        }
      }
    });

    var TDRecord = React.createClass({
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
            <tr><td>{this.props.t}</td><td>{data}</td></tr>
          );
        }
        return null;
      }
    });

    var URLCategory = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        if (!urldata.xfe.not_found) {
          var categories = Object.keys(mapOrUnknown(urldata.xfe.url_details.cats)).join(', ');
          if (categories) {
            return (
              <h3>Categories: {categories}</h3>
            );
          }
        }
        return null;
      }
    });

    var URLMalware = React.createClass({
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
            rows.push(<tr key={'mal_' + i}><td>{sorted[i].firstseen}</td><td>{sorted[i].type}</td><td>{sorted[i].md5}</td><td>{sorted[i].uri}</td><td>{arrOrUnknown(sorted[i].family).join(', ')}</td></tr>);
          }
          return (
            <div>
              <h3>Malware detected on URL</h3>
              <table className="table">
                <thead><th>First Seen</th><th>Type</th><th>MD5</th><th>URL</th><th>Family</th></thead>
                <tbody>{rows}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var URLVT = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        if (urldata && urldata.vt && urldata.vt.url_report && urldata.vt.url_report.response_code === 1) {
          var xfeFound = urldata.xfe && (!urldata.xfe.not_found || urldata.xfe.resolve && urldata.xfe.resolve.A);
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hurlvt">
                <h4 className="panel-title">
                  <a className={xfeFound ? 'collapsed' : ''} role="button" data-toggle="collapse" data-parent="#urlproviders" href="#urlvt" aria-expanded="true" aria-controls="urlvt">
                    Virus Total Data
                  </a>
                </h4>
              </div>
              <div id="urlvt" className={xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in'} role="tabpanel" aria-labelledby="hurlvt">
                <div className="panel-body">
                  <h4>
                    Scan Date: {urldata.vt.url_report.scan_date}
                    <br/>
                    Positives: {urldata.vt.url_report.positives}
                    <br/>
                    Total: {urldata.vt.url_report.total}
                  </h4>
                  <DetectedEngines urldata={urldata} />
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var DetectedEngines = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var detected_engines = [];
        var scans = urldata.vt.url_report.scans;
        for (var k in scans) {
          if (scans.hasOwnProperty(k) && scans[k].detected) {
            detected_engines.push(<tr key={'url_' + k}><td>{k}</td><td>{scans[k].result}</td></tr>);
          }
        }
        if (detected_engines.length > 0) {
          return (
            <div>
              <h4>Positive Detections</h4>
              <table>
                <thead><th>Scan Engine</th><th>Result</th></thead>
                <tbody>{detected_engines}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });
    // ======================== END URL section ===========================

    // ======================== File section ===========================
    var FileDiv = React.createClass({
      render: function() {
        if (!isfile && !ismd5) {
          return null;
        } else {
          var data = this.props.data;
          return (
            <div>
              <h2>File: {isfile ? data.file.details.name : data.md5s[0].details}</h2>
              <FileResultMessage data={data} />
              <FileAV data={data} />
              <FileDetails data={data.md5s[0]} />
            </div>
          );
        }
      }
    });

    var FileResultMessage = React.createClass({
      render: function() {
        var resultMessage = 'Could not determine the File reputation.';
        var color = 'warning-text';
        var filedata = this.props.data.file;
        var md5data = this.props.data.md5s[0];
        if (isfile && filedata.Result == 0 || ismd5 && md5data.Result == 0) {
          resultMessage = 'File is found to be clean.';
          color = 'success-text';
        } else if (isfile && filedata.Result == 1 || ismd5 && md5data.Result == 1) {
          resultMessage = 'File is found to be malicious.';
          color = 'danger-text';
        }
        return (<h3 className={color}>{resultMessage}</h3>);
      }
    });

    var FileAV = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data && data.file && data.file.virus) {
          return (<h2> Malware Name: {data.file.virus} </h2>);
        }
        return null;
      }
    });

    var FileDetails = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data &&
          (data.xfe && !data.xfe.not_found ||
          data.vt.file_report && data.vt.file_report.response_code === 1)) {
          return (
            <div className="panel-group" id="fileproviders" role="tablist" aria-multiselectable="true">
              <FileXFE data={data} />
              <FileVT data={data} />
            </div>
          );
        }
        return null;
      }
    });

    var FileXFE = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data && data.xfe && !data.xfe.not_found) {
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hfilexfe">
                <h4 className="panel-title">
                  <a role="button" data-toggle="collapse" data-parent="#fileproviders" href="#filexfe" aria-expanded="true" aria-controls="filexfe">
                    IBM X-Force Exchange Data
                  </a>
                </h4>
              </div>
              <div id="filexfe" className="panel-collapse collapse in" role="tabpanel" aria-labelledby="hfilexfe">
                <div className="panel-body">
                  <table className="table">
                    <tbody>
                      <tr><td>Type</td><td>{data.xfe.malware.type}</td></tr>
                      <tr><td>Mime Type</td><td>{data.xfe.malware.mimetype}</td></tr>
                      <tr><td>MD5</td><td>{data.xfe.malware.md5}</td></tr>
                      <tr><td>Family</td><td>{arrOrUnknown(data.xfe.malware.family).join(', ')}</td></tr>
                      <tr><td>Created</td><td>{data.xfe.malware.created}</td></tr>
                    </tbody>
                  </table>
                  <FileXFEOriginsEmail data={data.xfe.malware.origins.emails.rows} />
                  <FileXFEOriginsSubject data={data.xfe.malware.origins.subjects.rows} />
                  <FileXFEOriginsDown data={data.xfe.malware.origins.downloadServers.rows} />
                  <FileXFEOriginsCNC data={data.xfe.malware.origins.CnCServers.rows} />
                  <FileXFEOriginsExt data={data.xfe.malware.origins.external} />
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var FileXFEOriginsEmail = React.createClass({
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              <tr key={'file_email' + i}>
                <td>{data[i].firstseen}</td><td>{data[i].lastseen}</td><td>{data[i].origin}</td><td>{data[i].md5}</td><td>{data[i].filepath}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>Email Origins</h4>
              <table className="table">
                <thead><th>First Seen</th><th>Last Seen</th><th>Origin</th><th>MD5</th><th>File Path</th></thead>
                <tbody>{rows}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var FileXFEOriginsSubject = React.createClass({
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              <tr key={'file_subject' + i}>
                <td>{data[i].firstseen}</td><td>{data[i].lastseen}</td><td>{data[i].subject}</td><td>{data[i].ips ? data[i].ips.join(', ') : ''}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>Subjects</h4>
              <table className="table">
                <thead><th>First Seen</th><th>Last Seen</th><th>Subject</th><th>IPs</th></thead>
                <tbody>{rows}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var FileXFEOriginsDown = React.createClass({
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              <tr key={'file_down' + i}>
                <td>{data[i].firstseen}</td><td>{data[i].lastseen}</td><td>{data[i].host}</td><td>{data[i].uri}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>Download Servers</h4>
              <table className="table">
                <thead><th>First Seen</th><th>Last Seen</th><th>Host</th><th>URI</th></thead>
                <tbody>{rows}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var FileXFEOriginsCNC = React.createClass({
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.length > 0) {
          data.sort(sortByFirstSeen);
          for (var i=0; i < data.length && i < 10; i++) {
            rows.push(
              <tr key={'file_cnc' + i}>
                <td>{data[i].firstseen}</td><td>{data[i].lastseen}</td><td>{data[i].ip}</td><td>{arrOrUnknown(data[i].family).join(', ')}</td>
              </tr>
            )
          }
          return (
            <div>
              <h4>Command & Control Servers</h4>
              <table className="table">
                <thead><th>First Seen</th><th>Last Seen</th><th>IP</th><th>Family</th></thead>
                <tbody>{rows}</tbody>
              </table>
            </div>
          );
        }
        return null;
      }
    });

    var FileXFEOriginsExt = React.createClass({
      render: function() {
        var rows = [];
        var data = this.props.data;
        if (data && data.family && data.family.length > 0) {
          return (
            <div>
              <h4>External Detection</h4>
              <h5>{data.family.join(', ')}</h5>
            </div>
          );
        }
        return null;
      }
    });

    var FileVT = React.createClass({
      render: function() {
        var data = this.props.data;
        if (data && data.vt && data.vt.file_report && data.vt.file_report.response_code === 1) {
          var xfeFound = data.xfe && !data.xfe.not_found;
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="hfilevt">
                <h4 className="panel-title">
                  <a className={xfeFound ? 'collapsed' : ''} role="button" data-toggle="collapse" data-parent="#fileproviders" href="#filevt" aria-expanded="true" aria-controls="filevt">
                    Virus Total Data
                  </a>
                </h4>
              </div>
              <div id="filevt" className={xfeFound ? 'panel-collapse collapse' : 'panel-collapse collapse in'} role="tabpanel" aria-labelledby="hfilevt">
                <div className="panel-body">
                  <h4>
                    Scan Date: {data.vt.file_report.scan_date}
                    <br/>
                    Positives: {data.vt.file_report.positives}
                    <br/>
                    Total: {data.vt.file_report.total}
                  </h4>
                  <table className="table"><tbody>
                    <tr><td>MD5</td><td>{data.vt.file_report.md5}</td></tr>
                    <tr><td>SHA1</td><td>{data.vt.file_report.sha1}</td></tr>
                    <tr><td>SHA256</td><td>{data.vt.file_report.sha256}</td></tr>
                  </tbody></table>
                  <ScanResult data={data.vt.file_report} />
                </div>
              </div>
            </div>
          );
        }
        return null;
      }
    });

    var ScanResult = React.createClass ({
      render: function() {
        var scans = this.props.data.scans;
        var rows = [];
        for (var k in scans) {
          if (scans.hasOwnProperty(k) && scans[k].detected) {
            rows.push(<tr key={'fvt_scan_' + k}><td>{k}</td><td>{scans[k].result}</td><td>{scans[k].version}</td><td>{scans[k].update}</td></tr>);
          }
        }
        if (rows.length > 0) {
          return (
          <div>
            <h4>Detection Engines</h4>
            <table className="table">
              <thead>
                <th>Engine Name</th>
                <th>Version</th>
                <th>Result</th>
                <th>Update</th>
              </thead>
              <tbody>{rows}</tbody>
            </table>
          </div>
          );
        }
        return null;
      }
    });

    var DetailsDiv = React.createClass({
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
                <div>
                  <h2><center>D<small>BOT</small> is collecting security details for your query. It might take up to a minute!</center></h2>
                  <br/>
                  <br/>
                  <div className="ball-grid-pulse center-block">
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                  </div>

                </div>
              );
            } else if (this.state.status == 1) {
              return (
                <div>
                  <h1 className="text-center">D<small>BOT</small> Analysis Report</h1>
                  <URLDiv data={this.state.data} />
                  <FileDiv data={this.state.data} />
                  <IPDiv data={this.state.data} />
                </div>
              );
            }
            else {
              return(
                <div>
                  D<small>BOT</small> encountered an error while trying to serve your request. The issues has been reported and will be analyzed.
                  Please try to click the link again from Slack interface.
                  <hr></hr>
                  {this.state.errmsg}
                </div>
              );
            }
        }
    });

    React.render(<DetailsDiv />, document.getElementById('detailsdiv'));
  }
})(window.jQuery);

// END Details Handler
// -----------------------------------
