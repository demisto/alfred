import './IPDetails.less';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import autobind from 'autobind-decorator';
import { compareDate, dateToString, keysToString } from '../../utils/utils';
import Table from '../UIComponents/Table';
import { ReputationHeader } from './ReputationHeader';
import { ExpandableHeader } from '../UIComponents/ExpandableHeader';
import { HeaderSection } from './HeaderSection';

function foundXFE(xfe) {
  const { notFound } = xfe || {};
  return !notFound;
}

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
      expandXFE: true,
      expandVT: !foundXFE(props.xfe)
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

  getXFESection() {
    const { xfe } = this.props;
    const { expandXFE } = this.state;
    const { error, ipReputation, ipHistory } = xfe || {};
    if (!foundXFE(xfe)) {
      return null;
    }

    if (error) {
      return (
        <div className="error-message">
          {error}
        </div>
      );
    }

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
                  title="Subnets"
                  data={(ipReputation.subnets || []).sort(compareDate('created')).map(subnet => ({
                    ...subnet,
                    category: keysToString(subnet.cats),
                    location: (subnet.geo && subnet.geo.country) || 'Unknown',
                    created: dateToString(subnet.created)
                  }))}
                  keys={['subnet', 'score', 'category', 'location', 'reason', 'created']}
                />
                <Table
                  title="History"
                  data={(ipHistory && ipHistory.history  || []).sort(compareDate('created')).map(history => ({
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

    if (!vt) {
      return null;
    }

    const { error, ipReport } = vt;

    if (error) {
      return (
        <div className="error-message">
          {error}
        </div>
      );
    }

    const { Resolutions, detected_urls } = ipReport || {};
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
            {
              !(Resolutions && Resolutions.length > 0) && !(detected_urls && detected_urls.length > 0) &&
                <div className="h5 no-padding no-margin">
                  No results
                </div>
            }
            <div className="row">
              <Table
                title="Historical Resolutions"
                data={(Resolutions || []).sort(compareDate('last_resolved'))}
                keys={['hostname', 'last_resolved']}
                style={{ width: '40%' }}
              />
              <Table
                title="Detected URLs"
                data={(detected_urls || []).sort(compareDate('scan_date')).map(detected => ({
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
    return (
      <div className="ip-details">
        <h2>IP: {details}</h2>
        <ReputationHeader
          indicatorName="IP address"
          isPrivate={isPrivate}
          result={result}
        />
        {this.getXFESection()}
        {this.getVTSection()}
      </div>
    );
  }
}


export default IPDetails;
