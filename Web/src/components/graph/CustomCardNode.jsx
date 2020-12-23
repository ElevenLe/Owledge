import React from 'react';
import { Link } from "react-router-dom";
import { Card, Button} from 'antd';

function CardViewNode({data}){
    return (
        <div>
        <Card size="small" 
        title= {data.title}
        style={{ width: 300 }}>
        <p>{data.content}</p>
        </Card>
        </div>
    )
}

export default CardViewNode;