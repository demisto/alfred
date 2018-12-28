import '../semantic/dist/semantic.min.css';
import React, { Component } from 'react';
import './Details.css';
import { get } from './api';

class Details extends Component {

  constructor(props) {
    super(props);

    this.state = {
      status: '',
      loading: false,
      data: {}
    };
  }

  async componentDidMount() {
      const url = `/work${window.location.search}`;
      const { status, data } = await get(url);
      this.setState({ status, data })
  }

  render() {
    const { status, loading, data } = this.state;
    return (
      <div className="details-page ui grid">
        {(loading || true) &&
          <div className="ui segment">
            <div className="ui active dimmer">
              <div className="ui text loader">Loading</div>
            </div>
            <p>
              <h2>D<small>BOT</small> is collecting security details for your query. It might take up to a minute!</h2>
            </p>
          </div>
        }
      </div>
    );
  }
}

export default Details;
