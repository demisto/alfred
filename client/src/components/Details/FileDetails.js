import './FileDetails.less';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import autobind from 'autobind-decorator';
import { compareDate } from '../../utils/utils';
import Table from '../UIComponents/Table';
import { ReputationHeader } from './ReputationHeader';
import { ExpandableHeader } from '../UIComponents/ExpandableHeader';
import { HeaderSection } from './HeaderSection';

function foundXFE(xfe) {
  return xfe && !xfe.notFound;
}

function foundAF(af) {
  return af && !af.error;
}

function foundCY(cy) {
  return cy && !cy.error && cy.result && !cy.result.error;
}

class FileDetails extends Component {
  static propTypes = {
    details: PropTypes.string,
    result: PropTypes.number,
    xfe: PropTypes.object,
    vt: PropTypes.object,
    cy: PropTypes.object,
    af: PropTypes.object,
    file: PropTypes.object
  };

  constructor(props) {
    super(props);

    this.state = {
      expandAF: true,
      expandXFE: !foundAF(props.af),
      expandVT: !foundXFE(props.xfe) && !foundAF(props.af),
      expandCY: false
    };
  }

  @autobind
  onToggleXFE() {
    this.setState({ expandXFE: !this.state.expandXFE });
  }

  @autobind
  onToggleVT() {
    this.setState({ expandVT: !this.state.expandVT });
  }

  @autobind
  onToggleCY() {
    this.setState({ expandCY: !this.state.expandCY });
  }

  @autobind
  onToggleAF() {
    this.setState({ expandAF: !this.state.expandAF });
  }

  getXFESection() {
    const { xfe } = this.props;
    const { expandXFE } = this.state;
    if (!foundXFE(xfe)) {
      return null;
    }

    const { type, mimetype, md5, family, created, origins } = xfe.malware || {};

    const headers = [{
      label: 'Type:',
      value: type
    }, {
      label: 'Mime Type:',
      value: mimetype
    }, {
      label: 'MD5:',
      value: md5
    }, {
      label: 'Family:',
      value: (family || []).join(', ') || 'Unknown'
    }, {
      label: 'Created:',
      value: created
    }];

    const { emails, subjects, downloadServers, CnCServers, externals } = origins || {};

    return (
      <div className="ui left aligned grid">
        <div className="row">
          <ExpandableHeader
            title="IBM X-Force Exchange Data"
            expand={expandXFE}
            onClick={this.onToggleXFE}
          />
        </div>
        {
          expandXFE &&
            <div className="row ui left aligned padded grid">
              <HeaderSection
                headers={headers}
              />
              <div className="row">
                <Table
                  title="Email Origins"
                  data={((emails && emails.rows) || []).sort(compareDate('firstseen'))}
                  keys={['firstseen', 'lastseen', 'origin', 'md5', 'filepath']}
                  headers={['First Seen', 'Last Seen', 'Origin', 'MD5', 'File Path']}
                />
                <Table
                  title="Subjects"
                  data={((subjects && subjects.rows) || [])
                    .sort(compareDate('firstseen'))
                    .map( s => ({ ...s, ips: s.ips && s.ips.join(', ')}))
                  }
                  keys={['firstseen', 'lastseen', 'subject', 'ips',]}
                  headers={['First Seen', 'Last Seen', 'Subject', 'IPs']}
                />
                <Table
                  title="Download Servers"
                  data={((downloadServers && downloadServers.rows) || []).sort(compareDate('firstseen'))}
                  keys={['firstseen', 'lastseen', 'host', 'uri',]}
                  headers={['First Seen', 'Last Seen', 'Host', 'URI']}
                />
                <Table
                  title="Commands & Control Servers"
                  data={((CnCServers && CnCServers.rows) || [])
                    .sort(compareDate('firstseen'))
                    .map( s => ({ ...s, family: s.family && s.family.join(', ')}))
                  }
                  keys={['firstseen', 'lastseen', 'ip', 'family',]}
                  headers={['First Seen', 'Last Seen', 'IP', 'Family']}
                />
                {
                  externals && externals.family && externals.family.length > 0 &&
                    <div className="external-detection">
                      <div className="h4">External Detection</div>
                      <div className="h5">{externals.family.join(',')}</div>
                    </div>
                }
              </div>
            </div>
        }
      </div>
    )
  }

  getVTSection() {
    const { vt } = this.props;
    const { expandVT } = this.state;

    if (!vt) {
      return null;
    }

    const { fileReport } = vt;

    const headers = [{
      label: 'Scan Date:',
      value: (fileReport && fileReport.scan_date) || 'Unknown'
    }, {
      label: 'Positives:',
      value: (fileReport && fileReport.positives) || 0
    }, {
      label: 'Total:',
      value: (fileReport && fileReport.total) || 0
    }, {
      label: 'MD5:',
      value: (fileReport && fileReport.md5) || 'N/A'
    }, {
      label: 'SHA1:',
      value: (fileReport && fileReport.sha1) || 'N/A'
    }, {
      label: 'SHA256:',
      value: (fileReport && fileReport.sha256) || 'N/A'
    }];

    return (
      <div className="ui left aligned grid">
        <div className="row">
          <ExpandableHeader
            title="Virus Total Data"
            expand={expandVT}
            onClick={this.onToggleVT}
          />
        </div>
        {
          expandVT &&
          <div className="row ui left aligned padded grid">
            <HeaderSection
              headers={headers}
            />
            <div className="row">
              <Table
                title="Detection Engines"
                data={Object.entries((fileReport && fileReport.scans) || {})
                  .filter(([engine, scan]) => scan.detected)
                  .map(([engine, scan]) => ({
                    ...scan,
                    engine,
                  }))}
                keys={['engine', 'version', 'result', 'update']}
                headers={['Engine Name', 'Version','Result', 'Update']}
                style={{ width: '50%' }}
              />
            </div>
          </div>
        }
      </div>
    )
  }

  getAFSection() {
    const { af } = this.props;
    const { expandAF } = this.state;

    if (!foundAF(af)) {
      return null;
    }

    const { malware, tags, tag_groups, file_type, created, regions } = af.result || {};

    const headers = [{
      label: 'Malware:',
      value: malware.toString()
    }, {
      label: 'Created:',
      value: created
    }, {
      label: 'File Type:',
      value: file_type
    }, {
      label: 'Regions:',
      value: (regions || []).join(', ') || 'Unknown'
    }, {
    }, {
      label: 'Tags:',
      value: (tags || []).join(', ') || 'Unknown'
    }, {
      label: 'Tag Groups:',
      value: (tag_groups || []).join(', ') || 'Unknown'
    }];

    return (
        <div className="ui left aligned grid">
        <div className="row">
          <ExpandableHeader title="AutoFocus Data" expand={expandAF} onClick={this.onToggleAF} />
        </div>
        {
        expandAF &&
        <div className="row ui left aligned padded grid">
          <HeaderSection headers={headers} />
        </div>
        }
        </div>
    )
  }

  getCYSection() {
    const { cy } = this.props;
    const { expandCY } = this.state;

    if (!foundCY(cy)) {
      return null;
    }

    const { generalscore, classifiers } = cy.result || {};

    const headers = [{
      label: 'General Score:',
      value: generalscore
    }, {
      label: 'Classifiers:',
      value: ((Object.keys(classifiers) || []).reduce((acc, k) => acc + (acc === '' ? '' : ', ') + k + '(' + classifiers[k] + ')', '')) || 'Unknown'
    }];

    return (
        <div className="ui left aligned grid">
        <div className="row">
        <ExpandableHeader title="Cylance Infinity" expand={expandCY} onClick={this.onToggleCY} />
        </div>
        {
          expandCY &&
          <div className="row ui left aligned padded grid">
          <HeaderSection headers={headers} />
          </div>
        }
        </div>
    )
  }

  render() {
    const { details, result, file } = this.props;
    let readyDetails = details;
    let readyResult = result;
    if (file) {
      readyDetails = file.details && file.details.name;
      readyResult = file.result;
    }
    return (
      <div className="file-details">
        <div className="h4">File: {readyDetails}</div>
        <ReputationHeader
          indicatorName="File"
          result={readyResult}
        />
        {
          file && file.virus &&
            <div className="malware-name h5">Malware Name: {file.virus}</div>
        }
        {this.getAFSection()}
        {this.getXFESection()}
        {this.getVTSection()}
        {this.getCYSection()}
      </div>
    );
  }
}


export default FileDetails;
