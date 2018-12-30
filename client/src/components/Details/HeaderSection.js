import './HeaderSection.less';
import React from 'react';

export function HeaderSection({ headers }) {
  return (
    <div className="ui no-padding left aligned grid">
      {
        headers.map(({ label, value }) => (
          <div className="row no-padding">
            <div className="six wide bold column">{label}</div>
            <div className="ten wide column"> {value}</div>
          </div>
        ))
      }
    </div>
  );
}
