import React, { Component} from 'react';
import { Graph } from "react-d3-graph"

import CardViewNode from "./CustomCardNode"
import TreeGraph from "../treeGraph/treegraph"

import {API_ROOT} from "../constants"


const myConfig = {
    collapsible: true,
    height: 1000,
    width: 1000,
    d3: {
        linkLength: 200,
        gravity: -1000,
    },
    node: {
        color: "#d3d3d3",
        fontColor: "black",
        fontSize: 12,
        fontWeight: "normal",
        size: {
            width: 3000,
            height: 1000,
        },
        labelPosition: "top",
        labelProperty: "name",
        renderLabel: false,
        strokeColor: "none",
        strokeWidth: 1.5,
        viewGenerator: node => <CardViewNode data={node}/>,
    },
    link: {
        fontSize: 10,
        semanticStrokeWidth: true,
        strokeWidth: 1.5,
    }
}

class GraphPage extends Component{
    state = {
        graphData: {nodes: [{id:"1"}]},
    }
    onClickNode = nodeId => {
        this.props.moveToDetail(nodeId)
    }

    getGeneralInfo = () => {
        return fetch(`${API_ROOT}`, {
            method: "GET",
            headers: {
                Accept: 'application/json',
            }
        }).then((response) => {
            if (response.ok) {
                console.log("obtain general data from backend")
                return response.json()
            }
            throw new Error("general page: Fail to retrieve from server")
        }).then(data => {
            let graphData = data ? this.formGraphData(data.nodes) : []
            this.setState({
                graphData: graphData
            })
            
        }).catch(err => {
            console.log("General page: retrieve error", err)
            this.setState({
                err: "fetch general failed"
            })
        })
    }

    formGraphData = (data) => {
        let nodes = new Array()
        let links = new Array()
        for(var i = 0; i < data.length; i++){
            let node = {id: "", title: "", content: ""}
            let link = {source: "", target: ""}
            node.id = data[i].nodeID
            node.title = data[i].Title
            node.content = "this is" + data[i].Title + ". Please click me for more info"
            link.source = data[i].Parents
            link.target = data[i].nodeID

            nodes.push(node)
            if(link.source != ""){
                links.push(link)
            }
        }
        let passData = {
            nodes: nodes,
            links: links
        }
        return passData
    }

    render() {
        return (
            <div>
            <div>
            <TreeGraph />
            </div>
            <div className="Graph">
            <Graph
            id='test-graph' 
            data={this.state.graphData}
            config={myConfig}
            onClickNode={this.onClickNode}
            />
            </div>
            </div>
        )
    }

    componentWillMount(){
        this.getGeneralInfo()
    }
}

export default GraphPage

