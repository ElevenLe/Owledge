import React, {Component} from "react"
import {select, json, tree} from "d3"
const data = {
    name: "root",
    children: [
        {name: "children1",
         value: 100},
        {name: "children2",
         value: 100},
    ]
}

class treeGraph extends Component{
    
    render(){
        return (
            <div>
            </div>
        )
    }
}

export default treeGraph