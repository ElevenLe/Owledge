import React, {Component} from "react"
import { Switch, Route, Redirect } from "react-router-dom"
import {Layout} from "antd"
import Graph from "./graph/GraphPage"
import DetailMain from "./detailPage/detailMain"

const  {Header, Content} = Layout


class Main extends Component {
    state = {
        showGeneral: true,
        nodeId: 0,
    }
    render() {
        return (
            <Layout>
            <Content>
            <Switch>
                <Route path="/" render={this.getGeneral} />
            </Switch>
            </Content>
            </Layout>
        )
    }

    getGeneral = () => {
        return this.state.showGeneral ? <Graph moveToDetail={this.changeToLearnPage}/> 
                                      : <DetailMain moveToGeneral={this.changeToGeneralPage} nodeId={this.state.nodeId} />
    }

    changeToLearnPage = (node) => {
        this.setState({
            showGeneral : false,
            nodeId : node,
        })
    }

    changeToGeneralPage = () => {
        this.setState({
            showGeneral: true,
            nodeId: 0
        })
    }
}

export default Main;