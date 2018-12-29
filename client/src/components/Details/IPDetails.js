import './IPDetails.less';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { REPUTATION_RESULT } from '../../utils/constants';
import autobind from 'autobind-decorator';
import classNames from 'classnames';
import { dateToString, keysToString } from '../../utils/utils';
import sortBy from 'lodash/sortBy';
import Table from '../UIComponents/Table';

class IPDetails extends Component {
  static propTypes = {
    details: PropTypes.string,
    result: PropTypes.number,
    isPrivate: PropTypes.bool,
    xfe: PropTypes.object,
    vt: PropTypes.object
  };

  constructor(props) {
    super(props);

    this.state = {
      expandXFE: false,
      expandVT: false
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

  getHeader(isPrivate, result) {
    let headerMessage = 'Could not determine the IP address reputation.';
    let headerClass = 'unknown';

    if (isPrivate) {
      headerMessage = 'IP address is a private (internal) IP - no reputation found.';
      headerClass = 'clean';
    } else if (result === REPUTATION_RESULT.clean) {
      headerMessage = 'IP address is found to be clean.';
      headerClass = 'clean';
    } else if (result === REPUTATION_RESULT.dirty) {
      headerMessage = 'IP address is found to be malicious.';
      headerClass = 'dirty';
    }

    return (
      <div className={classNames('ui segment ip-reputation-header', headerClass)}>
        <h3>
          {headerMessage}
        </h3>
      </div>
    );
  }

  getXFESection() {
    const { xfe } = this.props;
    const { expandXFE } = this.state;
    const { notFound, error, ipReputation, ipHistory } = xfe || { notFound: true };
    if (notFound) {
      return null;
    }

    if (error) {
      return (
        <div className="error-message">
          {error}
        </div>
      );
    }

    const iconClass = classNames('chevron', { down: expandXFE, right: !expandXFE } , 'icon');

    const headers = [{
      label: 'Risk Score:',
      value: ipReputation.score
    }, {
      label: 'Country:',
      value: ipReputation.geo && ipReputation.geo.country || 'Unknown'
    }, {
      label: 'Categories:',
      value: keysToString(ipReputation.cats)
    }];


    return (
      <div className="ui left aligned grid">
        <div className="row vendor-toggle-button" onClick={this.onToggleXFE}>
          <i className={iconClass}/> IBM X-Force Exchange Data
        </div>
        {
          expandXFE &&
            <div className="row ui left aligned padded grid">
              {
                headers.map(({ label, value }) => (
                  <div className="row no-padding">
                    <div className="two wide bold column">{label}</div>
                    <div className="ten wide column"> {value}</div>
                  </div>
                ))
              }
              <div className="row">
                <Table
                  title="Subnets"
                  data={sortBy(ipReputation.subnets || [], 'created').map(subnet => ({
                    ...subnet,
                    category: keysToString(subnet.cats),
                    location: (subnet.geo && subnet.geo.country) || 'Unknown',
                    created: dateToString(subnet.created)
                  }))}
                  keys={['subnet', 'score', 'category', 'location', 'reason', 'created']}
                />
                <Table
                  title="History"
                  data={sortBy(ipHistory && ipHistory.history  || [], 'created').map(history => ({
                    ...history,
                    category: keysToString(history.cats),
                    location: (history.geo && history.geo.country) || 'Unknown',
                    created: dateToString(history.created)
                  }))}
                  keys={['ip', 'score', 'category', 'location', 'reason', 'created']}
                />
              </div>
            </div>
        }
      </div>
    )
  }

  getVTSection() {
    const { vt } = this.props;
    const { expandVT } = this.state;
    const { error, ipReport } = vt;

    if (error) {
      return (
        <div className="error-message">
          {error}
        </div>
      );
    }

    const iconClass = classNames('chevron', { down: expandVT, right: !expandVT } , 'icon');

    return (
      <div className="ui left aligned grid">
        <div className="row vendor-toggle-button" onClick={this.onToggleVT}>
          <i className={iconClass}/> Virus Total Data
        </div>
        {
          expandVT &&
          <div className="row ui left aligned padded grid">
            <div className="row">
              <Table
                title="Historical Resolutions"
                data={sortBy(ipReport && ipReport.Resolutions  || [], 'last_resolved')}
                keys={['hostname', 'last_resolved']}
                style={{ width: '40%' }}
              />
              <Table
                title="Detected URLs"
                data={sortBy(ipReport && ipReport.detected_urls  || [], 'scan_date').map(detected => ({
                  ...detected,
                  positives: `${detected.positives} / ${detected.total}`
                }))}
                keys={['url', 'positives', 'scan_date']}
                headers={['URL', 'Positives', 'Scan Date']}
                style={{ width: '70%' }}
              />
            </div>
          </div>
        }
      </div>
    )
  }



  render() {
    const { details, result, isPrivate } = this.props;
    const header = this.getHeader(isPrivate, result);
    return (
      <div className="ip-details">
        <h2>IP: {details}</h2>
        {header}
        {this.getXFESection()}
        {this.getVTSection()}
      </div>
    );
  }
}


export default IPDetails;
