import React, {Component} from "react"
import {Layout } from "antd"
import { Row, Col } from 'antd';
import {API_ROOT} from "../constants"
import MajorContent from "./MajorContent"
import Comment from "./Comment"

const { Header, Footer, Sider, Content } = Layout;

class DetailMain extends Component{
    state = {
        nodeId: 0,
        nodeContent: [],
        nodeComments: [],
        err: "",
    }


    loadNodeContent = (nodeId) => {
        return fetch(`${API_ROOT}/${nodeId}`, {
            method: "GET",
            headers: {
                Accept: 'application/json',
             },
        }).then((response) => {
            if (response.ok) {
                console.log("get data from backend")
                return response.json()
            }
            throw new Error("Failed to retrieve from server")
        })
        .then( data  => {
            this.setState({
                nodeContent: data.Node,
                nodeComments: data.Comments,
                nodeId: nodeId,
            })
        })
        .catch(err => {
            console.log("retrieve error", err)
            this.setState({
                isLoadingPosts: false,
                err: "fetch posts failed"
            })
        })
    }

    render(){
        const {nodeComments, nodeContent, nodeId} = this.state
        return(
            <div>
                <Row>
                    <Col span={8} className="nodePageSider"></Col>
                    <div className="majorContentContent">
                    <MajorContent nodeContent={nodeContent}/>
                    </div>
                    <div className="majorContentRelate"></div>
                    <Col span={8} className="nodePageSider"></Col>
                </Row>
                <Row>
                    <Col span={8} className="nodePageSider"></Col>
                    <div className="commentsArea">
                    <Comment loadNodeContent={this.loadNodeContent} nodeId={nodeId} nodeComments={nodeComments}></Comment>
                    </div>
                    <Col span={8} className="nodePageSider"></Col>
                </Row>
            </div>
           
        )
    }

    componentWillMount(){
        this.loadNodeContent(this.props.nodeId)
    }
}

export default DetailMain