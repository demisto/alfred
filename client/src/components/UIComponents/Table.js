import './Table.less';
import React from 'react';
import { fieldToTitle } from '../../utils/utils';

export default function({ title, headers, data, keys, style }) {
  if (!data || data.length === 0) {
    return null;
  }

  const readyHeaders = headers || keys.map(fieldToTitle);
  return (
    <div className="custom-table" style={style}>
      <h4>{title}</h4>
      <table className="ui celled table">
        <thead>
          {
            readyHeaders.map(header => (<th>{header}</th>))
          }
        </thead>
        <tbody>
        {
          data.map(((item, i) => (
            <tr key={i}>
              {keys.map(key => (<td>{item[key]}</td>))}
            </tr>
          )))
        }
        </tbody>
      </table>
    </div>
  )
}
