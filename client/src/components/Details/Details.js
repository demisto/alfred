import React, { Component } from 'react';
import './Details.css';
import { API_RESPONSE_STATUS } from '../../utils/constants';
import { parseType } from '../../utils/utils';
import { get } from '../../utils/api';
import IPDetails from "./IPDetails";
import URLDetails from './URLDetails';

class Details extends Component {

  constructor(props) {
    super(props);

    this.state = {
      status: '',
      loading: false,
      data: {}
    };
  }

  componentDidMount() {
      const url = `/work${window.location.search}`;
      this.setState({ loading: true }, async () => {
        const { status, data } = await get(url);
        this.setState({ status, data, loading: false })
      });
  }

  getDetailsSection(data) {
    const { type, ips, urls } = data;
    const { isIP, isURL, isFile, isMD5 } = parseType(type);
    return (
      <div className="ui centered grid">
        <div className="raw">
          <h3 className="text-center">D<small>BOT</small> Analysis Report:</h3>
        </div>
        <div className="row">
          {
            isIP && ips && ips[0] && <IPDetails {...ips[0]}/>
          }
          {
            isURL && urls && urls[0] && <URLDetails {...urls[0]}/>
          }
        </div>
      </div>
    );
  }

  render() {
    const { status, loading, data } = this.state;
    return (
      <div className="details-page">
        {loading &&
          <div className="ui centered grid">
            <div className="raw">
              <div className="ui active centered inline loader"></div>
            </div>
            <div className="row">
              <h2>D<small>BOT</small> is collecting security details for your query. It might take up to a minute!</h2>
            </div>
          </div>
        }
        {!loading && status === API_RESPONSE_STATUS.error &&
          <div className="ui centered grid">
            <div className="raw">
              <h1> OOps </h1>
              D<small>BOT</small> encountered an error while trying to serve your request. The issues has been reported and will be analyzed.
              Please try to click the link again from Slack interface.
            </div>
            <div className="row">
              Error details: { JSON.stringify(data, null, 2) }
            </div>
          </div>
        }

        {!loading && status === API_RESPONSE_STATUS.success && this.getDetailsSection(data)}
      </div>
    );
  }
}

export default Details;
